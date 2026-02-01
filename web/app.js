(() => {
  const $app = document.getElementById("app");

  const state = {
    search: {
      departure_station: "北京",
      arrival_station: "上海",
      travel_date: new Date().toISOString().slice(0, 10),
      train_type: "",
      seat_type: "",
      depart_time_start: "",
      depart_time_end: "",
      has_ticket: false,
      sort: "",
      direction: "asc",
      limit: 20,
      page: { index: 0, cursors: [""] },
      cursor: "",
    },
    trains: [],
    nextCursor: "",
    loading: false,
    virt: {
      scrollTop: 0,
      viewportHeight: 620,
      rowHeight: 132,
      overscan: 6,
      visibleTrainIds: [],
    },
    suggest: {
      dep: { items: [], open: false, seq: 0, activeIndex: -1, error: "" },
      arr: { items: [], open: false, seq: 0, activeIndex: -1, error: "" },
    },
    suggestTimer: { dep: null, arr: null },
    me: {
      token: localStorage.getItem("token") || "",
      user_id: localStorage.getItem("user_id") || "",
      phone: localStorage.getItem("phone") || "",
    },
    modal: null,
    toast: null,
    orders: [],
    orderDetail: { order_id: "", loading: false, data: null },
    profile: { loading: false, data: null, saving: false },
    passengers: { loading: false, items: [] },
    payDraft: null,
    payInfo: null,
    payPoll: null,
    chat: {
      open: false,
      msgs: [
        { role: "ai", content: "你好！我是你的智能购票助手。你可以直接对我说：“帮我查明天去上海的高铁”" }
      ],
      input: "",
      loading: false,
    },
    kb: {
      query: "",
      results: [],
      doc: { open: false, key: "", title: "", content: "", tags: "", saving: false }
    }
  };

  let renderScheduled = false;
  function renderSoon() {
    if (renderScheduled) return;
    renderScheduled = true;
    requestAnimationFrame(() => {
      renderScheduled = false;
      render();
    });
  }

  function setToast(msg, ms = 2200, type = "info") {
    if (typeof ms === "string") {
      type = ms;
      ms = 2200;
    }
    state.toast = { msg: String(msg || ""), type: String(type || "info") };
    renderSoon();
    if (ms > 0) setTimeout(() => {
      state.toast = null;
      renderSoon();
    }, ms);
  }

  function toastClassName(t) {
    const type = String(t?.type || "info").toLowerCase();
    if (type === "success" || type === "error" || type === "warn") return type;
    return "";
  }

  function orderStatusMeta(status) {
    const s = String(status || "").toUpperCase();
    if (s === "ISSUED") return { label: "已出票", tone: "ok" };
    if (s === "PAYING" || s === "PENDING_PAY") return { label: "待支付", tone: "warn" };
    if (s === "CANCELLED") return { label: "已取消", tone: "bad" };
    if (s === "REFUNDED") return { label: "已退票", tone: "bad" };
    if (s === "CHANGED") return { label: "已改签", tone: "info" };
    return { label: status || "-", tone: "info" };
  }

  function renderStatusTag(status) {
    const m = orderStatusMeta(status);
    return `<span class="status-tag ${htmlesc(m.tone)}">${htmlesc(m.label)}</span>`;
  }

  function htmlesc(s) {
    return String(s ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  async function api(method, path, body) {
    const headers = { "Content-Type": "application/json" };
    if (state.me.token) headers["Authorization"] = "Bearer " + state.me.token;
    const res = await fetch(path, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    const contentType = res.headers.get("content-type") || "";
    const isJson = contentType.includes("application/json");
    const data = isJson ? await res.json() : await res.text();
    if (res.status === 401) {
      logout();
      setHash("#/login?next=" + encodeURIComponent(currentHashForNext()));
      throw new Error("未登录或Token失效");
    }
    if (!res.ok) {
      const msg = typeof data === "string" ? data : (data?.msg || "请求失败");
      throw new Error(msg);
    }
    if (isJson && data && typeof data === "object" && Object.prototype.hasOwnProperty.call(data, "code")) {
      const code = Number(data.code);
      if (Number.isFinite(code) && code !== 200) {
        throw new Error(data.msg || "请求失败");
      }
    }
    return data;
  }

  function wsURL(path, params) {
    const u = new URL(path, location.origin);
    Object.entries(params || {}).forEach(([k, v]) => {
      if (v !== undefined && v !== null && String(v).trim() !== "") u.searchParams.set(k, String(v));
    });
    u.protocol = location.protocol === "https:" ? "wss:" : "ws:";
    return u.toString();
  }

  function createRemainWSManager() {
    const conns = new Map();
    const desired = new Set();
    const lastTime = new Map();
    let maxConns = 8;

    function keyOf(trainId, seatType, travelDate) {
      return `${trainId}|${seatType}|${travelDate}`;
    }

    function parseKey(key) {
      const [trainId, seatType, travelDate] = key.split("|");
      return { trainId, seatType, travelDate };
    }

    function applyRemainingUpdate(trainId, seatType, travelDate, remaining, tms) {
      const k = keyOf(trainId, seatType, travelDate);
      const prev = lastTime.get(k) || 0;
      if (tms && tms <= prev) return;
      if (tms) lastTime.set(k, tms);

      let changed = false;
      for (const it of state.trains) {
        if (it.train_id === trainId && String(it.seat_type || "") === String(seatType || "")) {
          if (it.remaining_seat_count !== remaining) {
            it.remaining_seat_count = remaining;
            changed = true;
          }
        }
      }
      const m = state.modal;
      if (m?.type === "buy" && m.train?.train_id === trainId && m.detail?.seat_types?.length) {
        for (const s of m.detail.seat_types) {
          if (s.seat_type === seatType) {
            if (s.remaining !== remaining) {
              s.remaining = remaining;
              changed = true;
            }
          }
        }
      }
      if (changed) renderSoon();
    }

    function connect(key) {
      if (conns.has(key)) return;
      const { trainId, seatType, travelDate } = parseKey(key);
      const obj = { ws: null, key, backoffMs: 500, timer: null, closedByUs: false };

      function scheduleReconnect() {
        if (!desired.has(key)) return;
        const wait = Math.min(10_000, obj.backoffMs);
        obj.backoffMs = Math.min(10_000, Math.floor(obj.backoffMs * 1.8));
        clearTimeout(obj.timer);
        obj.timer = setTimeout(() => {
          if (!desired.has(key)) return;
          open();
        }, wait + Math.floor(Math.random() * 250));
      }

      function open() {
        try {
          const url = wsURL("/api/v1/ticket/ws", { train_id: trainId, seat_type: seatType, travel_date: travelDate });
          const ws = new WebSocket(url);
          obj.ws = ws;
          obj.closedByUs = false;

          ws.addEventListener("open", () => {
            obj.backoffMs = 500;
          });
          ws.addEventListener("message", (ev) => {
            let data;
            try { data = JSON.parse(String(ev.data || "")); } catch { return; }
            if (data && typeof data === "object" && (data.remaining !== undefined)) {
              const tms = data.time ? Date.parse(data.time) : 0;
              applyRemainingUpdate(data.train_id || trainId, data.seat_type || seatType, data.travel_date || travelDate, Number(data.remaining || 0), tms);
            }
          });
          ws.addEventListener("close", () => {
            conns.delete(key);
            if (!obj.closedByUs) scheduleReconnect();
          });
          ws.addEventListener("error", () => {
            try { ws.close(); } catch {}
          });
        } catch {
          scheduleReconnect();
        }
      }

      obj.close = () => {
        obj.closedByUs = true;
        clearTimeout(obj.timer);
        try { obj.ws?.close(); } catch {}
      };

      conns.set(key, obj);
      open();
    }

    function enforceCap(prioritizedKeys) {
      const target = (prioritizedKeys || []).filter((k) => desired.has(k)).slice(0, maxConns);
      const keep = new Set(target);
      for (const k of Array.from(desired)) {
        if (!keep.has(k)) desired.delete(k);
      }
      for (const k of Array.from(conns.keys())) {
        if (!desired.has(k)) {
          const c = conns.get(k);
          c?.close?.();
          conns.delete(k);
        }
      }
      for (const k of Array.from(desired)) connect(k);
    }

    return {
      keyOf,
      setMax(n) { maxConns = Math.max(1, Math.min(20, Number(n || 8))); },
      sync(prioritizedKeys) {
        desired.clear();
        for (const k of prioritizedKeys || []) desired.add(k);
        enforceCap(prioritizedKeys || []);
      },
      closeAll() {
        desired.clear();
        for (const k of Array.from(conns.keys())) {
          const c = conns.get(k);
          c?.close?.();
        }
        conns.clear();
      },
    };
  }

  const remainWS = createRemainWSManager();

  function setHash(hash) {
    if (location.hash !== hash) location.hash = hash;
  }

  function parseHash() {
    const raw = location.hash || "#/";
    const idx = raw.indexOf("?");
    const path = idx >= 0 ? raw.slice(0, idx) : raw;
    const q = new URLSearchParams(idx >= 0 ? raw.slice(idx + 1) : "");
    return { raw, path, q };
  }

  function currentHashForNext() {
    return location.hash || "#/";
  }

  function loginOK(token, user_id, phone) {
    state.me.token = token;
    state.me.user_id = user_id || "";
    state.me.phone = phone || "";
    localStorage.setItem("token", state.me.token);
    localStorage.setItem("user_id", state.me.user_id);
    localStorage.setItem("phone", state.me.phone);
  }

  function logout() {
    state.me.token = "";
    state.me.user_id = "";
    state.me.phone = "";
    localStorage.removeItem("token");
    localStorage.removeItem("user_id");
    localStorage.removeItem("phone");
  }

  function formatMin(min) {
    const m = Number(min || 0);
    const h = Math.floor(m / 60);
    const mm = m % 60;
    return (h ? `${h}小时` : "") + `${mm}分`;
  }

  function validateRealName(name) {
    const s = String(name || "").trim();
    if (!s) return "请输入姓名";
    if (s.length > 20) return "姓名过长";
    if (!/^[\u4e00-\u9fa5·]{2,20}$/.test(s)) return "姓名需为2-20位中文字符（可含·）";
    return "";
  }

  function validateIDCard(idCard) {
    const s = String(idCard || "").trim().toUpperCase();
    if (!s) return "请输入身份证号";
    if (!/^\d{17}[\dX]$/.test(s)) return "身份证号格式错误";
    const weights = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2];
    const mapping = ["1", "0", "X", "9", "8", "7", "6", "5", "4", "3", "2"];
    let sum = 0;
    for (let i = 0; i < 17; i++) sum += Number(s[i]) * weights[i];
    const code = mapping[sum % 11];
    if (code !== s[17]) return "身份证校验位不正确";
    return "";
  }

  async function doSearch({ cursor = "" } = {}) {
    const dep = String(state.search.departure_station || "").trim();
    const arr = String(state.search.arrival_station || "").trim();
    if (!dep) { setToast("请输入出发地"); return; }
    if (!arr) { setToast("请输入到达地"); return; }
    if (dep === arr) { setToast("出发地与到达地不能相同"); return; }
    const today = new Date().toISOString().slice(0, 10);
    if (String(state.search.travel_date || "").trim() < today) { setToast("出行日期不能早于今天"); return; }

    const q = new URLSearchParams();
    q.set("departure_station", dep);
    q.set("arrival_station", arr);
    q.set("travel_date", state.search.travel_date);
    q.set("limit", String(state.search.limit || 20));
    if (state.search.train_type) q.set("train_type", state.search.train_type);
    if (state.search.seat_type) q.set("seat_type", state.search.seat_type);
    if (state.search.depart_time_start) q.set("depart_time_start", state.search.depart_time_start);
    if (state.search.depart_time_end) q.set("depart_time_end", state.search.depart_time_end);
    if (state.search.has_ticket) q.set("has_ticket", "true");
    if (state.search.sort) q.set("sort", state.search.sort);
    if (String(state.search.direction).toLowerCase() === "desc") q.set("direction", "desc");
    if (cursor) q.set("cursor", cursor);
    state.search.cursor = cursor || "";
    state.loading = true;
    renderSoon();
    try {
      const resp = await api("GET", "/api/v1/train/search?" + q.toString());
      const items = resp.items || [];
      state.trains = items;
      state.nextCursor = resp.next_cursor || "";
    } catch (e) {
      setToast(e.message || "查询失败");
    } finally {
      state.loading = false;
      renderSoon();
    }
  }

  async function loadOrders() {
    state.loading = true;
    renderSoon();
    try {
      const resp = await api("GET", "/api/v1/order/list?limit=20");
      state.orders = resp.orders || [];
    } catch (e) {
      setToast(e.message || "查询订单失败");
    } finally {
      state.loading = false;
      renderSoon();
    }
  }

  async function loadOrderDetail(orderId) {
    const id = String(orderId || "").trim();
    if (!id) return;
    state.orderDetail.order_id = id;
    state.orderDetail.loading = true;
    state.orderDetail.data = null;
    renderSoon();
    try {
      const q = new URLSearchParams();
      q.set("order_id", id);
      const resp = await api("GET", "/api/v1/order/info?" + q.toString());
      state.orderDetail.data = resp;
    } catch (e) {
      setToast(e.message || "加载订单失败");
    } finally {
      state.orderDetail.loading = false;
      renderSoon();
    }
  }

  async function loadProfile() {
    state.profile.loading = true;
    state.profile.data = null;
    renderSoon();
    try {
      const resp = await api("GET", "/api/v1/user/profile");
      state.profile.data = resp;
    } catch (e) {
      setToast(e.message || "加载个人信息失败");
    } finally {
      state.profile.loading = false;
      renderSoon();
    }
  }

  async function loadPassengers() {
    if (!state.me.token) return;
    state.passengers.loading = true;
    renderSoon();
    try {
      const resp = await api("GET", "/api/v1/user/passengers");
      state.passengers.items = resp.passengers || [];
    } catch (e) {
      state.passengers.items = [];
      setToast(e.message || "加载乘车人失败");
    } finally {
      state.passengers.loading = false;
      renderSoon();
    }
  }

  async function fetchPassengersSilently() {
    if (!state.me.token) return null;
    try {
      const resp = await api("GET", "/api/v1/user/passengers");
      state.passengers.items = resp.passengers || [];
      return resp;
    } catch {
      return null;
    }
  }

  function isRealNameVerified(profileResp) {
    const v = String(profileResp?.user_info?.real_name_verified || "").trim().toUpperCase();
    return v === "VERIFIED";
  }

  async function fetchProfileSilently() {
    if (!state.me.token) return null;
    try {
      const resp = await api("GET", "/api/v1/user/profile");
      state.profile.data = resp;
      return resp;
    } catch {
      return null;
    }
  }

  async function ensureBuyRuleVerified(meta) {
    if (!state.me.token) {
      setToast("请先登录");
      setHash("#/login?next=" + encodeURIComponent(meta?.next || currentHashForNext()));
      return false;
    }
    const prof = state.profile.data?.user_info ? state.profile.data : await fetchProfileSilently();
    if (isRealNameVerified(prof)) return true;
    setToast("请先完成实名认证");
    setHash("#/account?verify=1&next=" + encodeURIComponent(meta?.next || currentHashForNext()));
    return false;
  }

  function stopPayPolling() {
    if (state.payPoll?.timer) {
      clearInterval(state.payPoll.timer);
    }
    state.payPoll = null;
  }

  function startPayPolling(orderId) {
    stopPayPolling();
    const id = String(orderId || "").trim();
    if (!id) return;
    state.payPoll = {
      order_id: id,
      timer: setInterval(async () => {
        if (!state.me.token) return;
        try {
          const q = new URLSearchParams();
          q.set("order_id", id);
          const resp = await api("GET", "/api/v1/order/info?" + q.toString());
          state.payInfo = resp;
          const st = String(resp?.order?.order_status || "").trim().toUpperCase();
          if (st === "ISSUED" || st === "CANCELLED" || st === "REFUNDED" || st === "CHANGED") {
            stopPayPolling();
          }
          renderSoon();
        } catch {
        }
      }, 2000),
    };
  }

  function openBuyModal(item) {
    const user = state.profile.data?.user_info;
    const canSelf = isRealNameVerified(state.profile.data) && user?.real_name;
    const firstPassenger = canSelf
      ? { passenger_id: "", use_self: true, real_name: user.real_name || "", id_card: user.id_card || "", seat_type: "", seat_pref: "" }
      : { passenger_id: "", use_self: false, real_name: "", id_card: "", seat_type: "", seat_pref: "" };
    state.modal = {
      type: "buy",
      step: "passengers",
      train: item,
      detail: null,
      passengers: [firstPassenger],
      errors: {},
      creating: false,
    };
    renderSoon();
    fetchPassengersSilently();
    loadTrainDetail(item.train_id, item.departure_station, item.arrival_station);
  }

  function mountEvents() {
    document.querySelectorAll("[data-click]").forEach((el) => {
      el.addEventListener("click", (ev) => {
        const action = el.getAttribute("data-click");
        actions[action]?.(ev, el);
      });
    });
    document.querySelectorAll("[data-mousedown]").forEach((el) => {
      el.addEventListener("mousedown", (ev) => {
        ev.preventDefault();
        const action = el.getAttribute("data-mousedown");
        actions[action]?.(ev, el);
      });
    });
    document.querySelectorAll("[data-input]").forEach((el) => {
      const key = el.getAttribute("data-input");
      el.addEventListener("input", () => {
        if (el.type === "checkbox") state.search[key] = el.checked;
        else if (el.type === "number") state.search[key] = Number(el.value || 0);
        else state.search[key] = el.value;
      });
      el.addEventListener("change", () => {
        if (el.type === "checkbox") state.search[key] = el.checked;
        else if (el.type === "number") state.search[key] = Number(el.value || 0);
        else state.search[key] = el.value;
      });
    });
  }

  const actions = {
    navSearch() { setHash("#/"); },
    navTrains() { setHash("#/trains"); },
    navOrders() { setHash("#/orders"); },
    navMe() { setHash("#/account"); },
    navAccount() { setHash("#/account"); },
    navLogin() { setHash("#/login"); },
    reloadPassengers() { loadPassengers(); },
    reloadProfile() { loadProfile(); },
    openVerify() {
      const { q } = parseHash();
      const next = q.get("next");
      state.modal = {
        type: "verify_realname",
        real_name: "",
        id_card: "",
        phone: state.me.phone || "",
        submitting: false,
        errors: {},
        after: { next: next ? decodeURIComponent(next) : "#/" },
      };
      renderSoon();
    },
    useSelfPassenger(ev, el) {
      const idx = Number(el.getAttribute("data-idx"));
      const m = state.modal;
      if (!m || m.type !== "buy") return;
      const p = m.passengers[idx];
      if (!p) return;
      const user = state.profile.data?.user_info;
      if (!isRealNameVerified(state.profile.data) || !user?.real_name) {
        setToast("请先完成实名认证");
        return;
      }
      if (p.use_self) {
        p.use_self = false;
        p.real_name = "";
        p.id_card = "";
      } else {
        p.use_self = true;
        p.passenger_id = "";
        p.real_name = user.real_name || "";
        p.id_card = user.id_card || "";
      }
      renderSoon();
    },
    navOrder(ev, el) {
      const id = el.getAttribute("data-order-id");
      if (id) setHash("#/order/" + id);
    },
    doLogout() { logout(); setToast("已退出登录"); renderSoon(); },
    swapStations() {
      const a = state.search.departure_station;
      state.search.departure_station = state.search.arrival_station;
      state.search.arrival_station = a;
      renderSoon();
    },
    goTrains() {
      setHash("#/trains");
    },
    goHome() {
      setHash("#/");
    },
    doSearch() { doSearch({ cursor: state.search.cursor || "" }); },
    nextPage() {
      if (!state.nextCursor || state.loading) return;
      const p = state.search.page;
      const next = state.nextCursor;
      if (!p.cursors[p.index + 1]) p.cursors[p.index + 1] = next;
      p.index = Math.min(p.index + 1, p.cursors.length - 1);
      state.virt.scrollTop = 0;
      doSearch({ cursor: next });
    },
    prevPage() {
      const p = state.search.page;
      if (state.loading || p.index <= 0) return;
      p.index -= 1;
      state.virt.scrollTop = 0;
      doSearch({ cursor: p.cursors[p.index] || "" });
    },
    resetSearch() {
      state.trains = [];
      state.nextCursor = "";
      state.search.page = { index: 0, cursors: [""] };
      state.search.cursor = "";
      state.virt.scrollTop = 0;
      doSearch({ cursor: "" });
    },
    pickStation(ev, el) {
      const which = el.getAttribute("data-which");
      const value = el.getAttribute("data-value") || "";
      if (which === "dep") state.search.departure_station = value;
      if (which === "arr") state.search.arrival_station = value;
      state.suggest[which].open = false;
      render();
    },
    async openBuy(ev, el) {
      const id = el.getAttribute("data-train-id");
      const item = state.trains.find((t) => t.train_id === id);
      if (!item) return;
      const ok = await ensureBuyRuleVerified({ next: "#/order/confirm?train_id=" + encodeURIComponent(id) });
      if (!ok) return;
      setHash("#/order/confirm?train_id=" + encodeURIComponent(id));
    },
    async buyNext() {
      const m = state.modal;
      if (!m || m.type !== "buy") return;
      const ok = await ensureBuyRuleVerified({ next: "#/order/confirm?train_id=" + encodeURIComponent(m.train?.train_id || "") });
      if (!ok) return;
      const errs = {};
      for (let i = 0; i < m.passengers.length; i++) {
        const p = m.passengers[i];
        const e = {};
        if (!p.use_self && !String(p.passenger_id || "").trim()) {
          const nameMsg = validateRealName(p.real_name);
          const idMsg = validateIDCard(p.id_card);
          if (nameMsg) e.real_name = nameMsg;
          if (idMsg) e.id_card = idMsg;
        }
        if (!p.seat_type) e.seat_type = "请选择席别";
        if (Object.keys(e).length) errs[i] = e;
      }
      m.errors = errs;
      if (Object.keys(errs).length) {
        setToast("请检查实名信息与席别选择");
        renderSoon();
        return;
      }
      m.step = "confirm";
      renderSoon();
    },
    buyPrev() {
      const m = state.modal;
      if (!m || m.type !== "buy") return;
      m.step = "passengers";
      renderSoon();
    },
    closeModal() {
      const m = state.modal;
      if (m && (m.type === "buy" || m.type === "change")) {
        const shouldConfirm = m.type === "buy" ? true : Boolean(m.submitting);
        if (shouldConfirm) {
          const ok = window.confirm("确定要关闭吗？未完成的操作将丢失。");
          if (!ok) return;
        }
      }
      state.modal = null;
      const { path } = parseHash();
      if (path === "#/order/confirm") setHash("#/trains");
      else renderSoon();
    },
    async submitVerifyRealName() {
      const m = state.modal;
      if (!m || m.type !== "verify_realname") return;
      const real_name = String(m.real_name || "").trim();
      const id_card = String(m.id_card || "").trim();
      const phone = String(m.phone || "").trim();
      const errs = {};
      const nmsg = validateRealName(real_name);
      const imsg = validateIDCard(id_card);
      if (nmsg) errs.real_name = nmsg;
      if (imsg) errs.id_card = imsg;
      if (!/^1\d{10}$/.test(phone)) errs.phone = "手机号格式错误";
      m.errors = errs;
      if (Object.keys(errs).length) {
        renderSoon();
        return;
      }
      m.submitting = true;
      renderSoon();
      try {
        const resp = await api("POST", "/api/v1/user/verify_realname", { real_name, id_card, phone });
        if (resp?.code && resp.code !== 200) throw new Error(resp.msg || "实名认证失败");
        setToast("实名认证成功");
        await fetchProfileSilently();
        const after = m.after;
        state.modal = null;
        renderSoon();
        if (after?.next) setHash(after.next);
      } catch (e) {
        setToast(e.message || "实名认证失败");
      } finally {
        if (state.modal && state.modal.type === "verify_realname") {
          state.modal.submitting = false;
          renderSoon();
        }
      }
    },
    addPassenger() {
      state.modal.passengers.push({ passenger_id: "", use_self: false, real_name: "", id_card: "", seat_type: "", seat_pref: "" });
      renderSoon();
    },
    removePassenger(ev, el) {
      const idx = Number(el.getAttribute("data-idx"));
      state.modal.passengers.splice(idx, 1);
      if (state.modal.passengers.length === 0) state.modal.passengers.push({ passenger_id: "", use_self: false, real_name: "", id_card: "", seat_type: "", seat_pref: "" });
      renderSoon();
    },
    updatePassenger(ev, el) {
      const idx = Number(el.getAttribute("data-idx"));
      const field = el.getAttribute("data-field");
      const m = state.modal;
      if (!m || m.type !== "buy") return;
      const p = m.passengers[idx];
      if (!p) return;
      const prev = p[field];
      p[field] = el.value;
      if (field === "seat_type" && prev !== el.value) {
        p.seat_pref = "";
      }
      if (field === "passenger_id") {
        p.use_self = false;
        const pid = String(el.value || "").trim();
        if (!pid) {
          p.real_name = "";
          p.id_card = "";
          renderSoon();
          return;
        }
        const hit = (state.passengers.items || []).find((x) => String(x.passenger_id) === pid);
        if (hit) {
          p.real_name = hit.real_name || "";
          p.id_card = hit.id_card || "";
        }
        renderSoon();
      }
      if (field === "real_name" || field === "id_card") {
        p.use_self = false;
        p.passenger_id = "";
      }
    },
    pickSeatPref(ev, el) {
      const idx = Number(el.getAttribute("data-idx"));
      const seat = el.getAttribute("data-seat") || "";
      const m = state.modal;
      if (!m || m.type !== "buy") return;
      const p = m.passengers[idx];
      if (!p) return;
      p.seat_pref = p.seat_pref === seat ? "" : seat;
      for (let i = 0; i < m.passengers.length; i++) {
        if (i !== idx && m.passengers[i].seat_type === p.seat_type && m.passengers[i].seat_pref === p.seat_pref) {
          m.passengers[i].seat_pref = "";
        }
      }
      renderSoon();
    },
    async createOrder() {
      const m = state.modal;
      if (!m || m.type !== "buy") return;
      const ok = await ensureBuyRuleVerified({ next: "#/order/confirm?train_id=" + encodeURIComponent(m.train?.train_id || "") });
      if (!ok) return;
      const errs = {};
      for (let i = 0; i < m.passengers.length; i++) {
        const p = m.passengers[i];
        const e = {};
        if (!p.use_self && !String(p.passenger_id || "").trim()) {
          const nameMsg = validateRealName(p.real_name);
          const idMsg = validateIDCard(p.id_card);
          if (nameMsg) e.real_name = nameMsg;
          if (idMsg) e.id_card = idMsg;
        }
        if (!p.seat_type) e.seat_type = "请选择席别";
        if (Object.keys(e).length) errs[i] = e;
      }
      m.errors = errs;
      if (Object.keys(errs).length) {
        setToast("请检查实名信息与席别选择");
        renderSoon();
        return;
      }
      if (String(m.step || "") !== "confirm") {
        m.step = "confirm";
        renderSoon();
        return;
      }
      m.creating = true;
      renderSoon();
      try {
        const resp = await api("POST", "/api/v1/order/create", {
          train_id: m.train.train_id,
          departure_station: m.train.departure_station,
          arrival_station: m.train.arrival_station,
          passengers: m.passengers.map((p) => ({
            passenger_id: Number(p.passenger_id || 0) || 0,
            use_self: !!p.use_self,
            real_name: p.real_name,
            id_card: p.id_card,
            seat_type: p.seat_type,
          })),
        });
        if (resp?.seats?.length) {
          for (let i = 0; i < m.passengers.length; i++) {
            if (resp.seats[i]?.seat_num) m.passengers[i].seat_pref = resp.seats[i].seat_num;
          }
        }
        state.modal = null;
        state.payDraft = { order: resp, train: m.train, payResp: null };
        state.payInfo = null;
        setToast("下单成功，请支付");
        setHash("#/pay?order_id=" + encodeURIComponent(resp.order_id));
        startPayPolling(resp.order_id);
      } catch (e) {
        setToast(e.message || "下单失败");
      } finally {
        if (state.modal && state.modal.type === "buy") state.modal.creating = false;
        renderSoon();
      }
    },
    async payOrder() {
      const m = state.modal;
      if (!m || m.type !== "pay") return;
      m.paying = true;
      render();
      try {
        const resp = await api("POST", "/api/v1/order/pay", { order_id: m.order.order_id, pay_channel: m.payChannel || "ALIPAY" });
        m.payResp = resp;
        if (resp.pay_url) setToast("已生成支付链接");
      } catch (e) {
        setToast(e.message || "支付失败");
      } finally {
        m.paying = false;
        render();
      }
    },
    updatePayChannel(ev, el) {
      const m = state.modal;
      if (!m || m.type !== "pay") return;
      m.payChannel = el.value;
    },
    async refreshPayStatus() {
      const m = state.modal;
      if (!m || m.type !== "pay") return;
      try {
        const q = new URLSearchParams();
        q.set("order_id", m.order.order_id);
        const resp = await api("GET", "/api/v1/order/info?" + q.toString());
        m.orderInfo = resp?.order || null;
        if (m.orderInfo?.order_status) setToast("已刷新订单状态", 1200);
      } catch (e) {
        setToast(e.message || "刷新失败");
      } finally {
        render();
      }
    },
    openPayURL() {
      const m = state.modal;
      const url = m?.payResp?.pay_url;
      if (!url) return;
      window.open(url, "_blank", "noopener,noreferrer");
    },
    async mockPay() {
      const m = state.modal;
      if (!m || m.type !== "pay") return;
      const payNo = m.payResp?.pay_no;
      if (!payNo) { setToast("缺少pay_no，请先点支付生成"); return; }
      try {
        await api("POST", "/api/v1/pay/mock_notify", { order_id: m.order.order_id, pay_no: payNo, third_party_status: "SUCCESS" });
        setToast("已触发模拟回调");
      } catch (e) {
        setToast(e.message || "模拟回调失败");
      }
    },
    async cancelOrder(ev, el) {
      const id = el.getAttribute("data-order-id");
      try {
        await api("POST", "/api/v1/order/cancel", { order_id: id });
        setToast("已取消订单");
        loadOrders();
      } catch (e) {
        setToast(e.message || "取消失败");
      }
    },
    async refundOrder(ev, el) {
      const id = el.getAttribute("data-order-id");
      try {
        await api("POST", "/api/v1/order/refund", { order_id: id, reason: "前端发起退票" });
        setToast("已发起退票");
        loadOrders();
      } catch (e) {
        setToast(e.message || "退票失败");
      }
    },
    openChange(ev, el) {
      const id = el.getAttribute("data-order-id") || state.orderDetail.order_id;
      const od = state.orderDetail.data?.order;
      state.modal = {
        type: "change",
        order_id: id,
        travel_date: new Date().toISOString().slice(0, 10),
        departure_station: od?.departure_station || state.search.departure_station,
        arrival_station: od?.arrival_station || state.search.arrival_station,
        trains: [],
        loading: false,
        selected: null,
        resp: null,
      };
      render();
    },
    async changeSearch() {
      const m = state.modal;
      if (!m || m.type !== "change") return;
      const q = new URLSearchParams();
      q.set("departure_station", m.departure_station);
      q.set("arrival_station", m.arrival_station);
      q.set("travel_date", m.travel_date);
      q.set("limit", "20");
      m.loading = true;
      render();
      try {
        const resp = await api("GET", "/api/v1/train/search?" + q.toString());
        m.trains = resp.items || [];
      } catch (e) {
        setToast(e.message || "查询车次失败");
      } finally {
        m.loading = false;
        render();
      }
    },
    pickChangeTrain(ev, el) {
      const id = el.getAttribute("data-train-id");
      const m = state.modal;
      if (!m || m.type !== "change") return;
      const item = (m.trains || []).find((t) => t.train_id === id);
      if (!item) return;
      m.selected = item;
      render();
    },
    updateChangeField(ev, el) {
      const m = state.modal;
      if (!m || m.type !== "change") return;
      const field = el.getAttribute("data-field");
      m[field] = el.value;
    },
    async submitChange() {
      const m = state.modal;
      if (!m || m.type !== "change") return;
      if (!m.selected) {
        setToast("请选择要改签的车次");
        return;
      }
      m.submitting = true;
      render();
      try {
        const resp = await api("POST", "/api/v1/order/change", {
          order_id: m.order_id,
          new_train_id: m.selected.train_id,
          new_departure_station: m.selected.departure_station,
          new_arrival_station: m.selected.arrival_station,
        });
        m.resp = resp;
        setToast("改签成功", 2200, "success");
      } catch (e) {
        setToast(e.message || "改签失败", 2600, "error");
      } finally {
        m.submitting = false;
        render();
      }
    },
    async payPagePay(ev, el) {
      const { q } = parseHash();
      const orderId = el?.getAttribute?.("data-order-id") || q.get("order_id") || state.payDraft?.order?.order_id || "";
      if (!orderId) { setToast("缺少订单号"); return; }
      const channelEl = document.getElementById("pay_channel");
      const payChannel = String(channelEl?.value || "ALIPAY").trim() || "ALIPAY";
      try {
        const resp = await api("POST", "/api/v1/order/pay", { order_id: orderId, pay_channel: payChannel });
        if (state.payDraft) state.payDraft.payResp = resp;
        else state.payDraft = { order: { order_id: orderId }, train: null, payResp: resp };
        setToast(resp.pay_url ? "已生成支付链接" : "已发起支付");
        startPayPolling(orderId);
        renderSoon();
      } catch (e) {
        setToast(e.message || "支付失败");
      }
    },
    payPageOpen() {
      const url = state.payDraft?.payResp?.pay_url;
      if (!url) return;
      const w = window.open(url, "_blank", "noopener,noreferrer");
      if (!w) setToast("浏览器拦截了弹窗，请允许弹窗或手动复制 pay_url 打开", 2600, "warn");
    },
    copyPayURL() {
      const url = state.payDraft?.payResp?.pay_url || "";
      if (!url) { setToast("暂无 pay_url", 1800, "warn"); return; }
      (async () => {
        try {
          if (navigator?.clipboard?.writeText) {
            await navigator.clipboard.writeText(url);
            setToast("已复制 pay_url", 1400, "success");
            return;
          }
        } catch {}
        try {
          window.prompt("复制 pay_url", url);
        } catch {}
      })();
    },
    stopPayPoll() {
      stopPayPolling();
      setToast("已停止状态轮询", 1200, "info");
      renderSoon();
    },
    async payPageMock() {
      const { q } = parseHash();
      const orderId = q.get("order_id") || state.payDraft?.order?.order_id || "";
      const payNo = state.payDraft?.payResp?.pay_no;
      if (!orderId) { setToast("缺少订单号"); return; }
      if (!payNo) { setToast("缺少pay_no，请先发起支付"); return; }
      try {
        await api("POST", "/api/v1/pay/mock_notify", { order_id: orderId, pay_no: payNo, third_party_status: "SUCCESS" });
        setToast("已触发模拟回调");
        startPayPolling(orderId);
      } catch (e) {
        setToast(e.message || "模拟回调失败");
      }
    },
    async payPageRefresh() {
      const { q } = parseHash();
      const orderId = q.get("order_id") || state.payDraft?.order?.order_id || "";
      if (!orderId) { setToast("缺少订单号"); return; }
      try {
        const qq = new URLSearchParams();
        qq.set("order_id", orderId);
        const resp = await api("GET", "/api/v1/order/info?" + qq.toString());
        state.payInfo = resp;
        setToast("已刷新状态", 1200);
        renderSoon();
      } catch (e) {
        setToast(e.message || "刷新失败");
      }
    },
    async doLogin() {
      const phone = document.getElementById("login_phone").value.trim();
      const password = document.getElementById("login_password").value.trim();
      if (!phone || !password) { setToast("请输入手机号和密码"); return; }
      state.loading = true;
      renderSoon();
      try {
        const resp = await api("POST", "/api/v1/user/login", { phone, password });
        if (resp.code !== 200) throw new Error(resp.msg || "登录失败");
        loginOK(resp.token, resp.user_id, phone);
        setToast("登录成功");
        const { q } = parseHash();
        const next = q.get("next");
        setHash(next ? decodeURIComponent(next) : "#/");
        fetchProfileSilently();
        fetchPassengersSilently();
      } catch (e) {
        setToast(e.message || "登录失败");
      } finally {
        state.loading = false;
        renderSoon();
      }
    },
    async doRegister() {
      const user_name = document.getElementById("reg_user").value.trim();
      const phone = document.getElementById("reg_phone").value.trim();
      const password = document.getElementById("reg_password").value.trim();
      if (!user_name || !phone || !password) { setToast("请填写完整信息"); return; }
      state.loading = true;
      renderSoon();
      try {
        const resp = await api("POST", "/api/v1/user/register", { user_name, phone, password });
        if (resp.code !== 200) throw new Error(resp.msg || "注册失败");
        setToast("注册成功，请登录");
        setHash("#/login");
      } catch (e) {
        setToast(e.message || "注册失败");
      } finally {
        state.loading = false;
        renderSoon();
      }
    },
    async saveProfile() {
      const input = document.getElementById("profile_real_name");
      const real_name = (input?.value || "").trim();
      if (!real_name) { setToast("请输入姓名"); return; }
      state.profile.saving = true;
      render();
      try {
        const resp = await api("POST", "/api/v1/user/profile", { real_name });
        if (resp.code !== 200) throw new Error(resp.msg || "保存失败");
        state.profile.data = resp;
        setToast("保存成功");
      } catch (e) {
        setToast(e.message || "保存失败");
      } finally {
        state.profile.saving = false;
        render();
      }
    },
    toggleChat() {
      state.chat.open = !state.chat.open;
      renderSoon();
      if (state.chat.open) {
        setTimeout(() => {
          const el = document.getElementById("chat_input");
          if (el) el.focus();
          scrollToBottom();
        }, 100);
      }
    },
    updateChatInput(ev, el) {
      state.chat.input = el.value;
    },
    async sendChat() {
      const msg = String(state.chat.input || "").trim();
      if (!msg) return;
      if (!state.me.token) {
        state.chat.msgs.push({ role: "ai", content: "请先登录后再使用智能助手功能。" });
        state.chat.input = "";
        renderSoon();
        return;
      }
      state.chat.msgs.push({ role: "user", content: msg });
      state.chat.input = "";
      state.chat.loading = true;
      renderSoon();
      scrollToBottom();
      
      try {
        const resp = await api("POST", "/api/v1/assistant/chat", { message: msg });
        const reply = resp.reply || "（无回复）";
        const tool = resp.tool_calls?.[0];
        const newItem = { role: "ai", content: reply };
        if (tool) {
          newItem.tool = {
            name: tool.tool_name,
            args: tool.args,
            result: tool.result,
            error: tool.error
          };
          // 如果工具调用成功，且是搜索/下单类，可以在前端做一些跳转联动
          if (!tool.error && tool.tool_name === "SearchTrain") {
             // 自动跳转到列表页并应用搜索条件（可选体验优化）
             try {
               let args;
               if (typeof tool.args === "string") args = JSON.parse(tool.args);
               else args = tool.args;
               
               if (args.departure_station && args.arrival_station) {
                 state.search.departure_station = args.departure_station;
                 state.search.arrival_station = args.arrival_station;
                 if (args.date) state.search.travel_date = args.date;
                 // 延迟一下跳转，让用户看清回复
                 setTimeout(() => {
                   if (location.hash !== "#/trains") setHash("#/trains");
                   else doSearch();
                 }, 1500);
               }
             } catch {}
          }
        }
        state.chat.msgs.push(newItem);
      } catch (e) {
        state.chat.msgs.push({ role: "ai", content: "出错了：" + (e.message || "网络异常") });
      } finally {
        state.chat.loading = false;
        renderSoon();
        scrollToBottom();
      }
    },
    chatKeydown(ev, el) {
      if (ev.key === "Enter" && !ev.shiftKey) {
        ev.preventDefault();
        actions.sendChat();
      }
    },
    navKB() { setHash("#/kb"); },
    async kbSearch() {
      // 优先从 DOM 获取最新值，防止 state 不同步
      const inputEl = document.getElementById("kb_query_input");
      const domVal = inputEl ? inputEl.value : "";
      const q = String(domVal || state.kb.query || "").trim();
      
      console.log("[kbSearch] Query:", q);
      if (!q) { 
        state.kb.results = []; 
        setToast("请输入关键词");
        renderSoon(); 
        return; 
      }
      
      state.loading = true;
      renderSoon();
      try {
        setToast(`正在搜索: ${q}`, 1000); // 增加反馈
        const resp = await api("POST", "/api/v1/assistant/kb/search", { query: q, limit: 10 });
        console.log("[kbSearch] Resp:", resp);
        state.kb.results = resp.items || resp.Items || [];
        if (state.kb.results.length === 0) {
            setToast("未找到相关文档", 2000);
        }
      } catch (e) {
        console.error("[kbSearch] Error:", e);
        setToast(e.message || "搜索失败");
      } finally {
        state.loading = false;
        renderSoon();
      }
    },
    kbEdit(ev, el) {
      const idx = Number(el.getAttribute("data-idx"));
      const item = state.kb.results[idx];
      state.kb.doc = {
        open: true,
        key: item?.doc_key || "",
        title: item?.title || "",
        content: item?.content || "",
        tags: (item?.tags || []).join(","),
        saving: false
      };
      renderSoon();
    },
    kbNew() {
      state.kb.doc = {
        open: true,
        key: "",
        title: "",
        content: "",
        tags: "",
        saving: false
      };
      renderSoon();
    },
    kbClose() {
      state.kb.doc.open = false;
      renderSoon();
    },
    async kbSave() {
      const d = state.kb.doc;
      if (!d.key || !d.title || !d.content) { setToast("请填写完整（Key/标题/内容）"); return; }
      d.saving = true;
      renderSoon();
      try {
        await api("POST", "/api/v1/assistant/kb/upsert", {
          doc_key: d.key,
          title: d.title,
          content: d.content,
          tags: d.tags
        });
        setToast("保存成功");
        d.open = false;
        // 如果当前搜索框有内容，重新搜索刷新列表
        if (state.kb.query) actions.kbSearch();
      } catch (e) {
        setToast(e.message || "保存失败");
      } finally {
        d.saving = false;
        renderSoon();
      }
    },
    updateKBSearch(ev, el) { 
      state.kb.query = el.value; 
      // console.log("Input:", el.value); // Debug
    },
    updateKBField(ev, el) {
      const field = el.getAttribute("data-field");
      state.kb.doc[field] = el.value;
    }
  };

  function scrollToBottom() {
    const el = document.getElementById("chat_body");
    if (el) el.scrollTop = el.scrollHeight;
  }

  async function stationSuggest(which, keyword) {
    const kw = String(keyword || "").trim();
    if (!kw) {
      state.suggest[which].open = false;
      state.suggest[which].items = [];
      state.suggest[which].activeIndex = -1;
      state.suggest[which].error = "";
      renderSuggest(which);
      return;
    }
    const seq = ++state.suggest[which].seq;
    try {
      const q = new URLSearchParams();
      q.set("keyword", kw);
      q.set("limit", "10");
      const resp = await api("GET", "/api/v1/station/suggest?" + q.toString());
      if (state.suggest[which].seq !== seq) return;
      state.suggest[which].items = resp.items || [];
      state.suggest[which].open = true;
      state.suggest[which].error = "";
      state.suggest[which].activeIndex = state.suggest[which].items.length ? 0 : -1;
      renderSuggest(which);
    } catch {
      if (state.suggest[which].seq !== seq) return;
      state.suggest[which].open = true;
      state.suggest[which].items = [];
      state.suggest[which].activeIndex = -1;
      state.suggest[which].error = "网络异常";
      renderSuggest(which);
    }
  }

  function scheduleSuggest(which, keyword) {
    if (state.suggestTimer[which]) clearTimeout(state.suggestTimer[which]);
    state.suggestTimer[which] = setTimeout(() => stationSuggest(which, keyword), 250);
  }

  function renderSuggest(which) {
    const id = which === "dep" ? "dep_suggest" : "arr_suggest";
    const host = document.getElementById(id);
    if (!host) return;
    const box = state.suggest[which];
    const items = box?.items || [];
    if (box?.open) {
      host.style.display = "block";
      if (items.length) {
        const active = Number.isFinite(box.activeIndex) ? box.activeIndex : -1;
        host.innerHTML = items.map((s, idx) => {
          const cls = idx === active ? "suggest-item active" : "suggest-item";
          return `<div class="${cls}" data-value="${htmlesc(s)}" data-idx="${idx}">${htmlesc(s)}</div>`;
        }).join("");
      } else {
        const tip = box?.error ? box.error : "无匹配站点";
        host.innerHTML = `<div class="suggest-item" style="cursor:default;color:var(--muted);">${htmlesc(tip)}</div>`;
      }
    } else {
      host.style.display = "none";
      host.innerHTML = "";
    }
  }

  function wireSuggestInputs() {
    const dep = document.getElementById("dep_input");
    const arr = document.getElementById("arr_input");
    function bindKeydown(which, input) {
      if (!input) return;
      input.addEventListener("keydown", (ev) => {
        const box = state.suggest[which];
        if (!box?.open) return;
        const items = box.items || [];
        const hasItems = items.length > 0;
        if (ev.key === "Escape") {
          box.open = false;
          renderSuggest(which);
          return;
        }
        if (!hasItems) return;
        const max = items.length - 1;
        const cur = Number.isFinite(box.activeIndex) ? box.activeIndex : -1;
        if (ev.key === "ArrowDown") {
          ev.preventDefault();
          box.activeIndex = Math.min(max, Math.max(0, cur + 1));
          renderSuggest(which);
          return;
        }
        if (ev.key === "ArrowUp") {
          ev.preventDefault();
          box.activeIndex = Math.max(0, cur - 1);
          renderSuggest(which);
          return;
        }
        if (ev.key === "Enter") {
          const idx = Math.max(0, Math.min(max, cur < 0 ? 0 : cur));
          const v = items[idx] || "";
          if (!v) return;
          ev.preventDefault();
          if (which === "dep") state.search.departure_station = v;
          if (which === "arr") state.search.arrival_station = v;
          input.value = v;
          box.open = false;
          renderSuggest(which);
        }
      });
    }
    if (dep) {
      dep.addEventListener("input", () => {
        state.search.departure_station = dep.value;
        scheduleSuggest("dep", dep.value);
      });
      dep.addEventListener("focus", () => scheduleSuggest("dep", dep.value));
      dep.addEventListener("blur", () => setTimeout(() => {
        state.suggest.dep.open = false;
        renderSuggest("dep");
      }, 200));
      bindKeydown("dep", dep);
    }
    if (arr) {
      arr.addEventListener("input", () => {
        state.search.arrival_station = arr.value;
        scheduleSuggest("arr", arr.value);
      });
      arr.addEventListener("focus", () => scheduleSuggest("arr", arr.value));
      arr.addEventListener("blur", () => setTimeout(() => {
        state.suggest.arr.open = false;
        renderSuggest("arr");
      }, 200));
      bindKeydown("arr", arr);
    }

    const depSuggest = document.getElementById("dep_suggest");
    if (depSuggest) {
      depSuggest.addEventListener("mousedown", (ev) => {
        const it = ev.target?.closest?.(".suggest-item");
        if (!it) return;
        ev.preventDefault();
        const v = it.getAttribute("data-value") || "";
        state.search.departure_station = v;
        if (dep) dep.value = v;
        state.suggest.dep.open = false;
        renderSuggest("dep");
      });
    }
    const arrSuggest = document.getElementById("arr_suggest");
    if (arrSuggest) {
      arrSuggest.addEventListener("mousedown", (ev) => {
        const it = ev.target?.closest?.(".suggest-item");
        if (!it) return;
        ev.preventDefault();
        const v = it.getAttribute("data-value") || "";
        state.search.arrival_station = v;
        if (arr) arr.value = v;
        state.suggest.arr.open = false;
        renderSuggest("arr");
      });
    }
  }

  function wireVirtualList() {
    const el = document.getElementById("train_list_v");
    if (!el) return;

    if (Math.abs((el.scrollTop || 0) - (state.virt.scrollTop || 0)) > 1) el.scrollTop = state.virt.scrollTop || 0;

    el.addEventListener("scroll", () => {
      const st = el.scrollTop || 0;
      if (st === state.virt.scrollTop) return;
      state.virt.scrollTop = st;
      if (wireVirtualList._raf) return;
      wireVirtualList._raf = requestAnimationFrame(() => {
        wireVirtualList._raf = 0;
        render();
      });
    }, { passive: true });

    requestAnimationFrame(() => {
      const vh = el.clientHeight || 0;
      if (vh > 0 && Math.abs(vh - (state.virt.viewportHeight || 0)) > 2) {
        state.virt.viewportHeight = vh;
        render();
        return;
      }
      const row = el.querySelector("[data-virt-row]");
      if (!row) return;
      const rect = row.getBoundingClientRect();
      const cs = getComputedStyle(row);
      const mt = Number.parseFloat(cs.marginTop || "0") || 0;
      const mb = Number.parseFloat(cs.marginBottom || "0") || 0;
      const rh = rect.height + mt + mb;
      if (rh > 0 && Math.abs(rh - (state.virt.rowHeight || 0)) > 1) {
        state.virt.rowHeight = rh;
        render();
      }
    });
  }

  function syncRemainWS() {
    const keys = [];
    const date = state.search.travel_date;
    const m = state.modal;
    if (m?.type === "buy" && m.train?.train_id) {
      const set = new Set();
      for (const p of m.passengers || []) {
        const st = String(p.seat_type || "").trim();
        if (st) set.add(st);
      }
      for (const st of set) keys.push(remainWS.keyOf(m.train.train_id, st, date));
    }
    const hash = location.hash || "#/";
    if ((hash === "#/trains" || hash === "#/" || hash.startsWith("#/search")) && state.search.seat_type) {
      for (const id of state.virt.visibleTrainIds || []) {
        keys.push(remainWS.keyOf(id, state.search.seat_type, date));
      }
    }
    remainWS.setMax(8);
    remainWS.sync(keys);
  }

  async function loadTrainDetail(trainId, from, to) {
    const q = new URLSearchParams();
    q.set("train_id", trainId);
    q.set("departure_station", from);
    q.set("arrival_station", to);
    try {
      const resp = await api("GET", "/api/v1/train/detail?" + q.toString());
      if (state.modal && state.modal.type === "buy" && state.modal.train.train_id === trainId) {
        state.modal.detail = resp;
        if (resp.seat_types?.length) {
          const first = resp.seat_types.find((x) => x.remaining > 0) || resp.seat_types[0];
          if (first && state.modal.passengers[0] && !state.modal.passengers[0].seat_type) {
            state.modal.passengers.forEach((p) => (p.seat_type = first.seat_type));
          }
        }
        render();
      }
    } catch (e) {
      setToast(e.message || "加载车次详情失败");
    }
  }

    function viewTopbar(active) {
    const name = state.me.user_id ? `UID ${state.me.user_id.slice(0, 8)}…` : "未登录";
    const loginPart = state.me.token
      ? `<span class="pill">${htmlesc(name)}</span><button class="btn btn-ghost" data-click="doLogout">退出</button>`
      : `<button class="btn btn-ghost" data-click="navLogin">登录</button>`;
    
    // 增加 KB 入口
    const kbLink = state.me.token ? `<a href="#/kb" class="${active === "kb" ? "active" : ""}" data-click="navKB">知识库</a>` : "";

    return `
      <div class="topbar">
        <div class="topbar-inner">
          <div class="brand">携程买票 Demo</div>
          <div class="nav">
            <a href="#/" class="${active === "search" ? "active" : ""}" data-click="navSearch">查询</a>
            <a href="#/trains" class="${active === "trains" ? "active" : ""}" data-click="navTrains">列表</a>
            <a href="#/orders" class="${active === "orders" ? "active" : ""}" data-click="navOrders">订单</a>
            <a href="#/account" class="${active === "account" ? "active" : ""}" data-click="navAccount">账户</a>
            ${kbLink}
          </div>
          <div class="grow"></div>
          <div class="userbox">${loginPart}</div>
        </div>
      </div>
    `;
  }

  function viewHome() {
    const seatOptions = [
      { v: "", t: "不限席别" },
      { v: "硬座", t: "硬座" },
      { v: "二等座", t: "二等座" },
      { v: "一等座", t: "一等座" },
      { v: "商务座", t: "商务座" },
      { v: "硬卧", t: "硬卧" },
      { v: "软卧", t: "软卧" },
    ].map((o) => `<option value="${htmlesc(o.v)}" ${state.search.seat_type === o.v ? "selected" : ""}>${htmlesc(o.t)}</option>`).join("");

    const trainTypeOptions = [
      { v: "", t: "不限车次" },
      { v: "G", t: "高铁 G" },
      { v: "D", t: "动车 D" },
      { v: "Z", t: "直达 Z" },
      { v: "T", t: "特快 T" },
      { v: "K", t: "快速 K" },
    ].map((o) => `<option value="${htmlesc(o.v)}" ${state.search.train_type === o.v ? "selected" : ""}>${htmlesc(o.t)}</option>`).join("");

    return `
      ${viewTopbar("search")}
      <div class="container">
        <div class="card search-card">
          <div class="search-row">
            <div class="field station-field">
              <label>出发地</label>
              <input id="dep_input" class="input" value="${htmlesc(state.search.departure_station)}" placeholder="如 北京" autocomplete="off" />
              <div id="dep_suggest" class="suggest" style="display:none;"></div>
            </div>
            <button class="swap" title="交换" data-click="swapStations">⇄</button>
            <div class="field station-field">
              <label>到达地</label>
              <input id="arr_input" class="input" value="${htmlesc(state.search.arrival_station)}" placeholder="如 上海" autocomplete="off" />
              <div id="arr_suggest" class="suggest" style="display:none;"></div>
            </div>
            <div class="field">
              <label>出行日期</label>
              <input class="input" type="date" value="${htmlesc(state.search.travel_date)}" data-input="travel_date" />
            </div>
            <div class="field">
              <label>&nbsp;</label>
              <button class="btn btn-primary" data-click="goTrains" ${state.loading ? "disabled" : ""}>查询车次</button>
            </div>
          </div>
          <div class="search-row" style="grid-template-columns: 180px 180px 1fr 1fr 1fr; margin-top: 10px;">
            <div class="field">
              <label>车次类型</label>
              <select class="select" data-input="train_type">${trainTypeOptions}</select>
            </div>
            <div class="field">
              <label>席别</label>
              <select class="select" data-input="seat_type">${seatOptions}</select>
            </div>
            <div class="field">
              <label>出发时间起</label>
              <input class="input" type="time" value="${htmlesc(state.search.depart_time_start)}" data-input="depart_time_start" />
            </div>
            <div class="field">
              <label>出发时间止</label>
              <input class="input" type="time" value="${htmlesc(state.search.depart_time_end)}" data-input="depart_time_end" />
            </div>
            <div class="field">
              <label>只看有票</label>
              <label class="muted small" style="display:flex; align-items:center; gap:8px; padding:10px 0;">
                <input type="checkbox" ${state.search.has_ticket ? "checked" : ""} data-input="has_ticket" />
                仅返回余票>0
              </label>
            </div>
          </div>
          <div class="hint">提示：下单需要登录并完成实名认证（VERIFIED）。</div>
        </div>
      </div>
    `;
  }

  function viewSearch() {
    const items = state.trains || [];
    const rowH = Number(state.virt.rowHeight || 132);
    const vh = Number(state.virt.viewportHeight || 620);
    const over = Number(state.virt.overscan || 6);
    const start = Math.max(0, Math.floor((state.virt.scrollTop || 0) / rowH) - over);
    const end = Math.min(items.length, Math.ceil(((state.virt.scrollTop || 0) + vh) / rowH) + over);
    const topSpace = start * rowH;
    const bottomSpace = (items.length - end) * rowH;
    state.virt.visibleTrainIds = items.slice(start, end).map((t) => t.train_id);

    function renderTrainItem(t) {
      const remainClass = t.remaining_seat_count > 0 ? "ok" : "bad";
      const remainText = state.search.seat_type ? (t.remaining_seat_count > 0 ? `余票 ${t.remaining_seat_count}` : "无票") : "选择席别查看余票";
      const btnDisabled = state.search.seat_type && t.remaining_seat_count <= 0 ? "disabled" : "";
      return `
        <div class="train-item" data-virt-row="1">
          <div class="train-main">
            <div class="train-title">${htmlesc(t.train_id)} <span class="tag">${htmlesc(t.train_type)}</span></div>
            <div class="train-sub">${htmlesc(t.departure_station)} → ${htmlesc(t.arrival_station)} · ${formatMin(t.runtime_minutes)}</div>
          </div>
          <div>
            <div class="muted small">席别</div>
            <div>${htmlesc(t.seat_type || "-")}</div>
          </div>
          <div>
            <div class="price">¥${Number(t.seat_price || 0).toFixed(2)}</div>
            <div class="remain ${remainClass}">${remainText}</div>
          </div>
          <div style="display:flex;justify-content:flex-end;">
            <button class="btn btn-primary" ${btnDisabled} data-click="openBuy" data-train-id="${htmlesc(t.train_id)}">预订</button>
          </div>
        </div>
      `;
    }

    const skeletonList = `
      <div style="padding: 6px;">
        ${Array.from({ length: 6 }).map(() => {
          return `
            <div class="train-item" style="border-style:dashed;">
              <div class="train-main">
                <div class="skeleton skeleton-line lg" style="width: 220px;"></div>
                <div class="skeleton skeleton-line sm" style="width: 280px; margin-top: 10px;"></div>
              </div>
              <div>
                <div class="skeleton skeleton-line sm" style="width: 64px;"></div>
                <div class="skeleton skeleton-line" style="width: 120px; margin-top: 10px;"></div>
              </div>
              <div>
                <div class="skeleton skeleton-line lg" style="width: 120px;"></div>
                <div class="skeleton skeleton-line sm" style="width: 140px; margin-top: 10px;"></div>
              </div>
              <div style="display:flex;justify-content:flex-end;">
                <div class="skeleton" style="height: 40px; width: 120px;"></div>
              </div>
            </div>
          `;
        }).join("")}
      </div>
    `;

    const emptyState = `
      <div class="empty" style="margin: 12px 6px;">
        <h3>没有匹配的车次</h3>
        <p>试试调整日期/车次类型/席别，或取消“只看有票”。</p>
        <div style="display:flex; gap:10px; justify-content:center; margin-top: 12px; flex-wrap: wrap;">
          <button class="btn btn-secondary" data-click="swapStations">交换出发到达</button>
          <button class="btn btn-primary" data-click="resetSearch">重新搜索</button>
          <button class="btn btn-link" data-click="goHome">回到查询页</button>
        </div>
      </div>
    `;

    const listBody = state.loading
      ? skeletonList
      : (items.length
        ? `
          <div id="train_list_v" class="train-virt" style="height:${vh}px;">
            <div style="height:${topSpace}px;"></div>
            ${(items.slice(start, end).map(renderTrainItem).join(""))}
            <div style="height:${bottomSpace}px;"></div>
          </div>
        `
        : emptyState);

    const seatOptions = [
      { v: "", t: "不限席别" },
      { v: "硬座", t: "硬座" },
      { v: "二等座", t: "二等座" },
      { v: "一等座", t: "一等座" },
      { v: "商务座", t: "商务座" },
      { v: "硬卧", t: "硬卧" },
      { v: "软卧", t: "软卧" },
    ].map((o) => `<option value="${htmlesc(o.v)}" ${state.search.seat_type === o.v ? "selected" : ""}>${htmlesc(o.t)}</option>`).join("");

    const trainTypeOptions = [
      { v: "", t: "不限车次" },
      { v: "G", t: "高铁 G" },
      { v: "D", t: "动车 D" },
      { v: "Z", t: "直达 Z" },
      { v: "T", t: "特快 T" },
      { v: "K", t: "快速 K" },
    ].map((o) => `<option value="${htmlesc(o.v)}" ${state.search.train_type === o.v ? "selected" : ""}>${htmlesc(o.t)}</option>`).join("");

    return `
      ${viewTopbar("search")}
      <div class="container">
        <div class="card search-card">
          <div class="search-row">
            <div class="field station-field">
              <label>出发地</label>
              <input id="dep_input" class="input" value="${htmlesc(state.search.departure_station)}" placeholder="如 北京" autocomplete="off" />
              <div id="dep_suggest" class="suggest" style="display:none;"></div>
            </div>
            <button class="swap" title="交换" data-click="swapStations">⇄</button>
            <div class="field station-field">
              <label>到达地</label>
              <input id="arr_input" class="input" value="${htmlesc(state.search.arrival_station)}" placeholder="如 上海" autocomplete="off" />
              <div id="arr_suggest" class="suggest" style="display:none;"></div>
            </div>
            <div class="field">
              <label>出行日期</label>
              <input class="input" type="date" value="${htmlesc(state.search.travel_date)}" data-input="travel_date" />
            </div>
            <div class="field">
              <label>&nbsp;</label>
              <button class="btn btn-primary" data-click="resetSearch" ${state.loading ? "disabled" : ""}>搜索</button>
            </div>
          </div>
          <div class="search-row" style="grid-template-columns: 180px 180px 1fr 1fr 1fr; margin-top: 10px;">
            <div class="field">
              <label>车次类型</label>
              <select class="select" data-input="train_type">${trainTypeOptions}</select>
            </div>
            <div class="field">
              <label>席别</label>
              <select class="select" data-input="seat_type">${seatOptions}</select>
            </div>
            <div class="field">
              <label>出发时间起</label>
              <input class="input" type="time" value="${htmlesc(state.search.depart_time_start)}" data-input="depart_time_start" />
            </div>
            <div class="field">
              <label>出发时间止</label>
              <input class="input" type="time" value="${htmlesc(state.search.depart_time_end)}" data-input="depart_time_end" />
            </div>
            <div class="field">
              <label>排序</label>
              <div style="display:flex; gap:10px;">
                <select class="select" style="flex:1;" data-input="sort">
                  <option value="" ${state.search.sort ? "" : "selected"}>默认(发车时间)</option>
                  <option value="time" ${state.search.sort === "time" ? "selected" : ""}>按发车时间</option>
                  <option value="remain" ${state.search.sort === "remain" ? "selected" : ""}>按余票</option>
                  <option value="price" ${state.search.sort === "price" ? "selected" : ""} ${state.search.seat_type ? "" : "disabled"}>按价格${state.search.seat_type ? "" : "(需选席别)"}</option>
                </select>
                <select class="select" style="width:120px;" data-input="direction">
                  <option value="asc" ${String(state.search.direction).toLowerCase() !== "desc" ? "selected" : ""}>正序</option>
                  <option value="desc" ${String(state.search.direction).toLowerCase() === "desc" ? "selected" : ""}>倒序</option>
                </select>
              </div>
            </div>
          </div>
          <div style="display:flex; justify-content:space-between; align-items:center; margin-top: 10px;">
            <label class="muted small" style="display:flex; align-items:center; gap:8px;">
              <input type="checkbox" ${state.search.has_ticket ? "checked" : ""} data-input="has_ticket" />
              只看有票
            </label>
            <div class="muted small">提示：排序“按价格”仅在选择席别后才生效</div>
          </div>
          <div class="hint">提示：这是“携程风格”的 Demo 页面，接口对接网关 /api/v1。</div>
        </div>

        <div class="grid">
          <div class="card list">
            <div class="list-header">
              <div>车次列表</div>
              <div class="muted">
                ${items.length ? `第 ${state.search.page.index + 1} 页 · ${items.length} 条` : ""}
              </div>
            </div>
            ${listBody}
            <div style="display:flex;justify-content:space-between;gap:10px;padding:10px 6px;">
              <button class="btn btn-light" data-click="prevPage" ${state.loading || state.search.page.index <= 0 ? "disabled" : ""}>上一页</button>
              <div class="muted small" style="display:flex;align-items:center;gap:10px;">
                <span>每页</span>
                <select class="select" style="width:90px;padding:8px 10px;" data-input="limit">
                  ${[10,20,30,50].map((n) => `<option value="${n}" ${Number(state.search.limit || 20) === n ? "selected" : ""}>${n}</option>`).join("")}
                </select>
                <button class="btn btn-light" data-click="resetSearch" ${state.loading ? "disabled" : ""}>应用</button>
              </div>
              <button class="btn btn-light" data-click="nextPage" ${state.loading || !state.nextCursor ? "disabled" : ""}>下一页</button>
            </div>
          </div>
          <div class="card panel">
            <h3>流程提示</h3>
            <div class="muted small">1. 搜索车次 → 2. 预订（需要登录）→ 3. 下单锁座 → 4. 支付宝支付（生成 pay_url）→ 5. 支付回调推进出票</div>
            <div style="margin-top:10px;" class="muted small">本地调试：支付后可用“模拟回调”按钮推进订单状态。</div>
            <div class="actions">
              ${state.me.token ? `<button class="btn btn-light" data-click="navOrders">查看我的订单</button>` : `<button class="btn btn-light" data-click="navLogin">去登录</button>`}
            </div>
          </div>
        </div>
      </div>
    `;
  }

  function viewLogin() {
    return `
      ${viewTopbar("")}
      <div class="container">
        <div class="card panel">
          <h3>登录</h3>
          <div class="row">
            <div class="field">
              <label>手机号</label>
              <input id="login_phone" class="input" placeholder="13800138000" />
            </div>
            <div class="field">
              <label>密码</label>
              <input id="login_password" class="input" type="password" placeholder="密码" />
            </div>
          </div>
          <div class="actions">
            <button class="btn btn-primary" data-click="doLogin" ${state.loading ? "disabled" : ""}>登录</button>
            <button class="btn btn-light" data-click="navSearch">返回查询</button>
            <a href="#/register" class="btn btn-light">注册</a>
          </div>
          <div class="muted small" style="margin-top:10px;">登录后会自动在请求头携带 Authorization: Bearer token。</div>
        </div>
      </div>
    `;
  }

  function viewRegister() {
    return `
      ${viewTopbar("")}
      <div class="container">
        <div class="card panel">
          <h3>注册</h3>
          <div class="row">
            <div class="field">
              <label>用户名</label>
              <input id="reg_user" class="input" placeholder="u1" />
            </div>
            <div class="field">
              <label>手机号</label>
              <input id="reg_phone" class="input" placeholder="13800138000" />
            </div>
          </div>
          <div class="row" style="margin-top:10px;">
            <div class="field">
              <label>密码</label>
              <input id="reg_password" class="input" type="password" placeholder="密码" />
            </div>
            <div></div>
          </div>
          <div class="actions">
            <button class="btn btn-primary" data-click="doRegister" ${state.loading ? "disabled" : ""}>注册</button>
            <a href="#/login" class="btn btn-light">去登录</a>
          </div>
        </div>
      </div>
    `;
  }

  function viewPay() {
    const { q } = parseHash();
    const orderId = q.get("order_id") || state.payDraft?.order?.order_id || "";
    if (!state.me.token) {
      return `
        ${viewTopbar("")}
        <div class="container">
          <div class="card panel">
            <h3>订单支付</h3>
            <div class="muted">需要先登录</div>
            <div class="actions"><button class="btn btn-primary" data-click="navLogin">去登录</button></div>
          </div>
        </div>
      `;
    }

    const info = state.payInfo?.order || null;
    const payUrl = state.payDraft?.payResp?.pay_url || "";
    const payNo = state.payDraft?.payResp?.pay_no || "";
    const st = String(info?.order_status || "").trim() || "-";
    const deadline = info?.pay_deadline_unix ? new Date(info.pay_deadline_unix * 1000).toLocaleString() : "-";
    const isSandbox = payUrl.includes("openapi.alipaydev.com");
    const polling = Boolean(state.payPoll);

    return `
      ${viewTopbar("")}
      <div class="container">
        <div class="card panel">
          <h3>订单支付</h3>
          <div class="muted small">订单ID：${htmlesc(orderId || "-")}</div>
          <div class="card" style="padding:12px;margin-top:12px;">
            <div class="row">
              <div>
                <div class="muted small">状态</div>
                <div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap;">
                  ${renderStatusTag(st)}
                  ${polling ? `<span class="tag" style="background:rgba(22,119,255,0.10);color:#1d4ed8;">状态轮询中</span>` : ``}
                </div>
              </div>
              <div><div class="muted small">支付截止</div><div>${htmlesc(deadline)}</div></div>
            </div>
          </div>
          ${isSandbox ? `
            <div class="banner warn" style="margin-top:12px;">
              当前为支付宝沙箱环境，网关偶发 502/不可用属于外部环境问题。
              如果收银台打不开，可使用“模拟回调成功”先验证下单→支付→出票链路。
            </div>
          ` : ``}
          <div style="margin-top:10px;display:flex;gap:10px;align-items:center;">
            <div class="muted small">支付渠道</div>
            <select id="pay_channel" class="select" style="width:200px;padding:8px 10px;">
              <option value="ALIPAY">支付宝</option>
              <option value="WECHAT">微信支付</option>
              <option value="CARD">银行卡</option>
            </select>
          </div>
          <div class="actions">
            <button class="btn btn-primary" data-click="payPagePay" data-order-id="${htmlesc(orderId)}">发起支付</button>
            <button class="btn btn-secondary" data-click="payPageOpen" ${payUrl ? "" : "disabled"}>打开收银台</button>
            <button class="btn btn-secondary" data-click="payPageMock" ${state.payDraft?.payResp?.pay_no ? "" : "disabled"}>模拟回调成功</button>
            <button class="btn btn-secondary" data-click="payPageRefresh">刷新状态</button>
            <button class="btn btn-secondary" data-click="stopPayPoll" ${polling ? "" : "disabled"}>停止轮询</button>
            <button class="btn btn-link" data-click="navOrders">订单中心</button>
          </div>
          ${payUrl ? `
            <div class="card" style="padding:12px;margin-top:12px;">
              <div class="row">
                <div>
                  <div class="muted small">pay_no</div>
                  <div style="word-break:break-all;">${htmlesc(payNo || "-")}</div>
                </div>
                <div style="display:flex; justify-content:flex-end; align-items:end; gap:10px; flex-wrap: wrap;">
                  <button class="btn btn-secondary" data-click="copyPayURL">复制 pay_url</button>
                  <button class="btn btn-secondary" data-click="payPageOpen">新窗口打开</button>
                </div>
              </div>
              <div style="margin-top:10px;" class="muted small">pay_url</div>
              <div style="word-break:break-all;">${htmlesc(payUrl)}</div>
            </div>
          ` : ``}
        </div>
      </div>
    `;
  }

  function viewOrders() {
    if (!state.me.token) {
      return `
        ${viewTopbar("orders")}
        <div class="container">
          <div class="card panel">
            <h3>我的订单</h3>
            <div class="muted">需要先登录</div>
            <div class="actions"><button class="btn btn-primary" data-click="navLogin">去登录</button></div>
          </div>
        </div>
      `;
    }

    const rows = (state.orders || []).map((o) => {
      const st = String(o.order_status || "").toUpperCase();
      const canCancel = st === "PENDING_PAY" || st === "PAYING";
      const canRefund = st === "ISSUED";
      const canPay = st === "PENDING_PAY" || st === "PAYING";
      return `
      <tr>
        <td><a href="#/order/${htmlesc(o.order_id)}" data-click="navOrder" data-order-id="${htmlesc(o.order_id)}">${htmlesc(o.order_id)}</a></td>
        <td>${htmlesc(o.train_id)}</td>
        <td>${htmlesc(o.departure_station)} → ${htmlesc(o.arrival_station)}</td>
        <td>¥${Number(o.total_amount || 0).toFixed(2)}</td>
        <td>${renderStatusTag(o.order_status || "-")}</td>
        <td>
          ${canPay ? `<a class="btn btn-primary" href="#/pay?order_id=${encodeURIComponent(o.order_id)}">去支付</a>` : ``}
          <button class="btn btn-secondary" data-click="cancelOrder" data-order-id="${htmlesc(o.order_id)}" ${canCancel ? "" : "disabled"}>取消</button>
          <button class="btn btn-secondary" data-click="refundOrder" data-order-id="${htmlesc(o.order_id)}" ${canRefund ? "" : "disabled"}>退票</button>
        </td>
      </tr>
    `;
    }).join("");

    return `
      ${viewTopbar("orders")}
      <div class="container">
        <div class="card panel">
          <h3>我的订单</h3>
          <div class="actions" style="margin-top:0;">
            <button class="btn btn-primary" data-click="reloadOrders" ${state.loading ? "disabled" : ""}>刷新</button>
            <button class="btn btn-light" data-click="navSearch">去查询</button>
          </div>
          <div style="margin-top:10px;overflow:auto;">
            <table class="table">
              <thead>
                <tr>
                  <th>订单ID</th>
                  <th>车次</th>
                  <th>区间</th>
                  <th>金额</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>${rows || `<tr><td colspan="6" class="muted">${state.loading ? "加载中…" : "暂无订单"}</td></tr>`}</tbody>
            </table>
          </div>
        </div>
      </div>
    `;
  }

  function viewMe() {
    if (!state.me.token) {
      return `
        ${viewTopbar("account")}
        <div class="container">
          <div class="card panel">
            <h3>账户与实名</h3>
            <div class="muted">需要先登录</div>
            <div class="actions"><button class="btn btn-primary" data-click="navLogin">去登录</button></div>
          </div>
        </div>
      `;
    }

    const loading = state.profile.loading;
    const user = state.profile.data?.user_info;
    const verified = String(user?.real_name_verified || "").toUpperCase() === "VERIFIED";
    const passengers = state.passengers.items || [];
    const passengerRows = passengers.length
      ? passengers.map((p) => `<tr><td>${htmlesc(p.real_name || "-")}</td><td>${htmlesc(p.id_card || "-")}</td></tr>`).join("")
      : `<tr><td colspan="2" class="muted">${state.passengers.loading ? "加载中…" : "暂无常用乘车人"}</td></tr>`;

    return `
      ${viewTopbar("account")}
      <div class="container">
        <div class="card panel">
          <h3>账户与实名</h3>
          ${loading ? `<div class="muted">加载中…</div>` : ""}
          ${user ? `
            <div class="card" style="padding:12px;margin-top:12px;">
              <div class="row">
                <div><div class="muted small">用户ID</div><div>${htmlesc(user.user_id || "-")}</div></div>
                <div><div class="muted small">手机号</div><div>${htmlesc(user.phone || "-")}</div></div>
              </div>
              <div class="row" style="margin-top:10px;">
                <div><div class="muted small">实名状态</div><div>${htmlesc(user.real_name_verified || "-")}</div></div>
                <div><div class="muted small">证件号</div><div>${htmlesc(user.id_card || "-")}</div></div>
              </div>
            </div>

            <div class="card" style="padding:12px;margin-top:12px;">
              <div class="field">
                <label>姓名（未实名可修改）</label>
                <input id="profile_real_name" class="input" value="${htmlesc(user.real_name || "")}" ${verified ? "disabled" : ""} />
              </div>
              <div class="actions">
                <button class="btn btn-primary" data-click="saveProfile" ${verified || state.profile.saving ? "disabled" : ""}>保存</button>
                ${verified ? `` : `<button class="btn btn-primary" data-click="openVerify">去实名</button>`}
                <button class="btn btn-light" data-click="reloadProfile" ${state.profile.loading ? "disabled" : ""}>刷新</button>
                <button class="btn btn-light" data-click="reloadPassengers" ${state.passengers.loading ? "disabled" : ""}>刷新乘车人</button>
                <button class="btn btn-light" data-click="navTrains">去购票</button>
              </div>
              ${verified ? `<div class="muted small" style="margin-top:8px;">已实名用户姓名不可在此修改。</div>` : ``}
            </div>

            <div class="card" style="padding:12px;margin-top:12px;">
              <div style="display:flex;justify-content:space-between;align-items:center;">
                <div><strong>常用乘车人</strong></div>
                <div class="muted small">下单页可直接选择</div>
              </div>
              <div style="margin-top:10px;overflow:auto;">
                <table class="table">
                  <thead><tr><th>姓名</th><th>证件</th></tr></thead>
                  <tbody>${passengerRows}</tbody>
                </table>
              </div>
            </div>
          ` : (!loading ? `<div class="muted">暂无数据</div>` : "")}
        </div>
      </div>
    `;
  }

  function viewOrderDetail(orderId) {
    if (!state.me.token) {
      return `
        ${viewTopbar("")}
        <div class="container">
          <div class="card panel">
            <h3>订单详情</h3>
            <div class="muted">需要先登录</div>
            <div class="actions"><button class="btn btn-primary" data-click="navLogin">去登录</button></div>
          </div>
        </div>
      `;
    }

    const loading = state.orderDetail.loading;
    const data = state.orderDetail.data;
    const order = data?.order;
    const seats = data?.seats || [];
    const seatRows = seats.map((s) => `
      <tr>
        <td>${htmlesc(s.seat_type)}</td>
        <td>${htmlesc(s.carriage_num)}</td>
        <td>${htmlesc(s.seat_num)}</td>
        <td>¥${Number(s.seat_price || 0).toFixed(2)}</td>
      </tr>
    `).join("");

    const st = String(order?.order_status || "").toUpperCase();
    const canCancel = st === "PENDING_PAY" || st === "PAYING";
    const canRefund = st === "ISSUED";
    const canPay = st === "PENDING_PAY" || st === "PAYING";

    return `
      ${viewTopbar("orders")}
      <div class="container">
        <div class="card panel">
          <h3>订单详情</h3>
          <div class="muted small">订单ID：${htmlesc(orderId)}</div>
          ${loading ? `<div class="muted" style="margin-top:10px;">加载中…</div>` : ""}
          ${order ? `
            <div class="card" style="padding:12px;margin-top:12px;">
              <div class="row">
                <div><div class="muted small">区间</div><div>${htmlesc(order.departure_station)} → ${htmlesc(order.arrival_station)}</div></div>
                <div><div class="muted small">金额</div><div>¥${Number(order.total_amount || 0).toFixed(2)}</div></div>
              </div>
              <div class="row" style="margin-top:10px;">
                <div><div class="muted small">状态</div><div>${renderStatusTag(order.order_status || "-")}</div></div>
                <div><div class="muted small">支付截止</div><div>${new Date((order.pay_deadline_unix || 0) * 1000).toLocaleString()}</div></div>
              </div>
            </div>
            <div class="actions">
              <button class="btn btn-light" data-click="navOrders">返回订单列表</button>
              ${canPay ? `<a class="btn btn-primary" href="#/pay?order_id=${encodeURIComponent(orderId)}">去支付</a>` : ``}
              <button class="btn btn-secondary" data-click="cancelOrder" data-order-id="${htmlesc(orderId)}" ${canCancel ? "" : "disabled"}>取消</button>
              <button class="btn btn-secondary" data-click="refundOrder" data-order-id="${htmlesc(orderId)}" ${canRefund ? "" : "disabled"}>退票</button>
              <button class="btn btn-primary" data-click="openChange" data-order-id="${htmlesc(orderId)}">改签</button>
            </div>
            <div style="margin-top:12px;overflow:auto;">
              <table class="table">
                <thead><tr><th>席别</th><th>车厢</th><th>座位</th><th>价格</th></tr></thead>
                <tbody>${seatRows || `<tr><td colspan="4" class="muted">暂无席位信息</td></tr>`}</tbody>
              </table>
            </div>
          ` : (!loading ? `<div class="muted" style="margin-top:12px;">订单不存在或无权限访问</div>` : "")}
        </div>
      </div>
    `;
  }

  actions.reloadOrders = () => loadOrders();

  function viewKB() {
    if (!state.me.token) {
      return `
        ${viewTopbar("kb")}
        <div class="container">
          <div class="card panel">
            <h3>知识库管理</h3>
            <div class="muted">需要先登录</div>
            <div class="actions"><button class="btn btn-primary" data-click="navLogin">去登录</button></div>
          </div>
        </div>
      `;
    }
    
    const rows = state.kb.results.map((d, idx) => `
      <tr>
        <td>${htmlesc(d.doc_key)}</td>
        <td>${htmlesc(d.title)}</td>
        <td><div class="muted small" style="max-width:300px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">${htmlesc(d.content)}</div></td>
        <td>${(d.tags || []).map(t => `<span class="tag">${htmlesc(t)}</span>`).join(" ")}</td>
        <td>
          <button class="btn btn-secondary" data-click="kbEdit" data-idx="${idx}">编辑</button>
        </td>
      </tr>
    `).join("");

    return `
      ${viewTopbar("kb")}
      <div class="container">
        <div class="card panel">
          <div style="display:flex;justify-content:space-between;align-items:center;">
            <h3>知识库管理</h3>
            <button class="btn btn-primary" data-click="kbNew">新建文档</button>
          </div>
          <div class="row" style="margin-top:16px;">
            <div class="field">
              <input class="input" placeholder="输入关键词搜索文档..." value="${htmlesc(state.kb.query)}" data-input="kb_search" />
            </div>
            <div style="flex:0 0 80px;">
              <button class="btn btn-primary" style="width:100%;" data-click="kbSearch" ${state.loading ? "disabled" : ""}>搜索</button>
            </div>
          </div>
          
          <div style="margin-top:16px;overflow:auto;">
            <table class="table">
              <thead><tr><th>Key</th><th>标题</th><th>内容摘要</th><th>标签</th><th>操作</th></tr></thead>
              <tbody>
                ${rows || `<tr><td colspan="5" class="muted" style="text-align:center;padding:20px;">${state.loading ? "加载中..." : (state.kb.query ? "无匹配结果" : "请输入关键词搜索")}</td></tr>`}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    `;
  }

  function viewModal() {
    const m = state.modal;
    if (state.kb.doc.open) {
       const d = state.kb.doc;
       return `
        <div class="modal-mask" data-click="kbClose">
          <div class="modal" onclick="event.stopPropagation()">
            <div class="modal-head">
              <div><strong>${d.key ? "编辑文档" : "新建文档"}</strong></div>
              <button class="close" data-click="kbClose">关闭</button>
            </div>
            <div class="modal-body">
              <div class="row">
                <div class="field">
                  <label>文档唯一键 (Key)</label>
                  <input class="input" value="${htmlesc(d.key)}" data-field="key" placeholder="如 refund_policy_v1" ${d.saving ? "disabled" : ""} />
                </div>
                <div class="field">
                  <label>标题</label>
                  <input class="input" value="${htmlesc(d.title)}" data-field="title" placeholder="如 退票规则" ${d.saving ? "disabled" : ""} />
                </div>
              </div>
              <div class="field" style="margin-top:10px;">
                <label>正文内容</label>
                <textarea class="input" style="height:120px;resize:vertical;" data-field="content" placeholder="输入文档正文..." ${d.saving ? "disabled" : ""}>${htmlesc(d.content)}</textarea>
              </div>
              <div class="field" style="margin-top:10px;">
                <label>标签 (逗号分隔)</label>
                <input class="input" value="${htmlesc(d.tags)}" data-field="tags" placeholder="如 规则,退票" ${d.saving ? "disabled" : ""} />
              </div>
              <div class="actions">
                <button class="btn btn-secondary" data-click="kbClose" ${d.saving ? "disabled" : ""}>取消</button>
                <button class="btn btn-primary" data-click="kbSave" ${d.saving ? "disabled" : ""}>保存</button>
              </div>
            </div>
          </div>
        </div>
       `;
    }

    if (!m) return "";

    if (m.type === "verify_realname") {
      const e = m.errors || {};
      const phoneHint = (String(state.me.phone || "").trim() && String(m.phone || "").trim() && String(m.phone || "").trim() !== String(state.me.phone || "").trim())
        ? `<div class="banner warn" style="margin-top:10px;">手机号需与当前账号一致（当前账号：${htmlesc(state.me.phone)}）。</div>`
        : "";
      return `
        <div class="modal-mask" data-click="closeModal">
          <div class="modal" onclick="event.stopPropagation()">
            <div class="modal-head">
              <div><strong>实名认证</strong></div>
              <button class="close" data-click="closeModal">关闭</button>
            </div>
            <div class="modal-body">
              <div class="banner info">购票规则：未实名认证用户不可下单。请提交姓名 + 身份证 + 当前账号手机号。</div>
              ${phoneHint}
              <div class="row" style="margin-top:10px;">
                <div class="field">
                  <label>姓名</label>
                  <input id="verify_real_name" class="input ${e.real_name ? "invalid" : ""}" value="${htmlesc(m.real_name || "")}" placeholder="如 张三" />
                  ${e.real_name ? `<div class="error">${htmlesc(e.real_name)}</div>` : ""}
                </div>
                <div class="field">
                  <label>身份证号</label>
                  <input id="verify_id_card" class="input ${e.id_card ? "invalid" : ""}" value="${htmlesc(m.id_card || "")}" placeholder="18位身份证号" />
                  ${e.id_card ? `<div class="error">${htmlesc(e.id_card)}</div>` : ""}
                </div>
              </div>
              <div class="row" style="margin-top:10px;">
                <div class="field">
                  <label>手机号</label>
                  <input id="verify_phone" class="input ${e.phone ? "invalid" : ""}" value="${htmlesc(m.phone || "")}" placeholder="11位手机号" />
                  ${e.phone ? `<div class="error">${htmlesc(e.phone)}</div>` : ""}
                </div>
                <div class="field">
                  <label>&nbsp;</label>
                  <div class="muted small">手机号需要与当前账号一致</div>
                </div>
              </div>
              <div class="actions">
                <button class="btn btn-secondary" data-click="closeModal" ${m.submitting ? "disabled" : ""}>取消</button>
                <button class="btn btn-primary" data-click="submitVerifyRealName" ${m.submitting ? "disabled" : ""}>提交认证</button>
              </div>
            </div>
          </div>
        </div>
      `;
    }

    if (m.type === "buy") {
      if (!m.step) m.step = "passengers";
      const step = String(m.step || "passengers");
      const seatOptions = (m.detail?.seat_types || []).map((s) => {
        const label = `${s.seat_type}（余${s.remaining} · ¥${Number(s.min_price || 0).toFixed(2)}起）`;
        const disabled = s.remaining <= 0 ? "disabled" : "";
        return `<option value="${htmlesc(s.seat_type)}" ${disabled}>${htmlesc(label)}</option>`;
      }).join("");

      const passengerOptions = (state.passengers.items || []).map((x) => {
        const label = `${x.real_name || ""} ${x.id_card || ""}`.trim();
        return `<option value="${htmlesc(x.passenger_id)}">${htmlesc(label || String(x.passenger_id))}</option>`;
      }).join("");

      const errs = m.errors || {};

      const seatTypes = m.detail?.seat_types || [];
      const seatTypeRemain = new Map();
      for (const s of seatTypes) seatTypeRemain.set(String(s.seat_type || "").trim(), Number(s.remaining || 0) || 0);
      const needByType = new Map();
      for (const p of (m.passengers || [])) {
        const st = String(p.seat_type || "").trim();
        if (!st) continue;
        needByType.set(st, (needByType.get(st) || 0) + 1);
      }
      const shortageTypes = [];
      for (const [st, need] of needByType.entries()) {
        const rem = seatTypeRemain.get(st) || 0;
        if (rem < need) shortageTypes.push(`${st} 余票 ${rem} < 需要 ${need}`);
      }
      const seatDetailHint = m.detail ? "" : `<div class="banner info" style="margin-top:10px;">车次详情加载中（席别/余票/票价）…</div>`;
      const ps = m.passengers.map((p, idx) => {
        const e = errs[idx] || {};
        const isSelf = !!p.use_self;
        const locked = (isSelf || String(p.passenger_id || "").trim()) ? "disabled" : "";
        const disableSelect = isSelf ? "disabled" : "";
        const selectedSeatType = String(p.seat_type || "").trim();
        const remainEntry = (m.detail?.seat_types || []).find((x) => x.seat_type === selectedSeatType);
        const remain = remainEntry ? Number(remainEntry.remaining || 0) : 0;
        const need = selectedSeatType ? (needByType.get(selectedSeatType) || 0) : 0;
        const shortage = selectedSeatType && remain > 0 && need > remain;
        const seatLabel = p.seat_pref ? `偏好：${p.seat_pref}` : "偏好：自动分配";
        const cols = ["A", "B", "", "C", "D"];
        let seatButtons = "";
        if (selectedSeatType) {
          let seatIdx = 0;
          const rows = 10;
          for (let pos = 0; pos < rows * cols.length; pos++) {
            const row = Math.floor(pos / cols.length) + 1;
            const col = cols[pos % cols.length];
            if (!col) {
              seatButtons += `<div class="seat-gap"></div>`;
              continue;
            }
            seatIdx += 1;
            const seat = `${String(row).padStart(2, "0")}${col}`;
            const isSelected = p.seat_pref === seat;
            const isDisabled = remain > 0 ? seatIdx > remain : true;
            const cls = "seat" + (isSelected ? " selected" : "") + (isDisabled ? " disabled" : "");
            seatButtons += `<button class="${cls}" ${isDisabled ? "disabled" : ""} data-click="pickSeatPref" data-idx="${idx}" data-seat="${htmlesc(seat)}">${htmlesc(seat)}</button>`;
          }
        } else {
          seatButtons = `<div class="muted small">先选择席别后可选座（演示用）</div>`;
        }
        return `
        <div class="card" style="padding:12px;margin-top:10px;">
          <div class="row">
            <div class="field">
              <label>乘车人（常用）</label>
              <select class="select" ${disableSelect} data-idx="${idx}" data-field="passenger_id" data-click="noop">
                <option value="">手动填写</option>
                ${passengerOptions}
              </select>
            </div>
            <div style="display:flex;align-items:end;justify-content:flex-end;gap:10px;">
              <button class="btn btn-secondary" data-click="useSelfPassenger" data-idx="${idx}">本人</button>
              <button class="btn btn-secondary" data-click="reloadPassengers" ${state.passengers.loading ? "disabled" : ""}>刷新</button>
              <button class="btn btn-secondary" data-click="removePassenger" data-idx="${idx}" ${m.passengers.length <= 1 ? "disabled" : ""}>删除</button>
            </div>
          </div>
          <div class="row">
            <div class="field">
              <label>乘客姓名</label>
              <input class="input" ${locked} value="${htmlesc(p.real_name)}" data-idx="${idx}" data-field="real_name" data-click="noop" />
              ${e.real_name ? `<div class="error">${htmlesc(e.real_name)}</div>` : ""}
            </div>
            <div class="field">
              <label>身份证号</label>
              <input class="input" ${locked} value="${htmlesc(p.id_card)}" data-idx="${idx}" data-field="id_card" data-click="noop" />
              ${e.id_card ? `<div class="error">${htmlesc(e.id_card)}</div>` : ""}
            </div>
          </div>
          <div class="row" style="margin-top:10px;">
            <div class="field">
              <label>席别</label>
              <select class="select" ${m.detail ? "" : "disabled"} data-idx="${idx}" data-field="seat_type" data-click="noop">
                <option value="">请选择</option>
                ${seatOptions}
              </select>
              ${e.seat_type ? `<div class="error">${htmlesc(e.seat_type)}</div>` : ""}
            </div>
            <div style="display:flex;align-items:end;justify-content:flex-end;"></div>
          </div>
          <div style="margin-top:10px;">
            <div class="muted small">${htmlesc(seatLabel)}${selectedSeatType ? ` · 实时余票 ${remain}` : ""}</div>
            ${shortage ? `<div class="banner warn" style="margin-top:10px;">${htmlesc(selectedSeatType)} 余票不足（${remain}），当前选择人数为 ${need}。</div>` : ""}
            <div class="seat-map">${seatButtons}</div>
          </div>
        </div>
      `;
      }).join("");

      const stepper = `
        <div class="stepper">
          <div class="step ${step === "passengers" ? "active" : "done"}">1 乘车人</div>
          <div class="step ${step === "confirm" ? "active" : ""}">2 确认订单</div>
          <div class="step">3 支付出票</div>
        </div>
      `;

      const priceBySeatType = new Map();
      for (const s of (m.detail?.seat_types || [])) {
        const k = String(s.seat_type || "").trim();
        if (!k) continue;
        priceBySeatType.set(k, Number(s.min_price || 0) || 0);
      }
      let estimate = 0;
      for (const p of (m.passengers || [])) {
        const st = String(p.seat_type || "").trim();
        estimate += priceBySeatType.get(st) || 0;
      }
      estimate = Number(estimate.toFixed(2));
      const dateText = state.search.travel_date || new Date().toISOString().slice(0, 10);
      const passengerSummary = (m.passengers || []).map((p) => {
        const name = String(p.real_name || "").trim() || "-";
        const st = String(p.seat_type || "").trim() || "-";
        const idc = String(p.id_card || "").trim();
        const masked = idc.length >= 10 ? (idc.slice(0, 4) + "****" + idc.slice(-4)) : idc;
        const price = priceBySeatType.get(st);
        return `
          <div class="card" style="padding:10px;margin-top:10px;border-radius:14px;box-shadow:none;">
            <div class="row">
              <div><div class="muted small">乘客</div><div>${htmlesc(name)}</div></div>
              <div><div class="muted small">证件</div><div>${htmlesc(masked || "-")}</div></div>
            </div>
            <div class="row" style="margin-top:8px;">
              <div><div class="muted small">席别</div><div>${htmlesc(st)}</div></div>
              <div><div class="muted small">预估票价</div><div>${price !== undefined ? `¥${Number(price).toFixed(2)}起` : "-"}</div></div>
            </div>
          </div>
        `;
      }).join("");

      const body = step === "confirm"
        ? `
          <div class="card" style="padding:12px;margin-top:10px;">
            <div class="row">
              <div><div class="muted small">出行日期</div><div>${htmlesc(dateText)}</div></div>
              <div><div class="muted small">预估总价</div><div class="price" style="font-size:16px;">¥${estimate.toFixed(2)}起</div></div>
            </div>
            <div class="muted small" style="margin-top:8px;">说明：座位与金额以锁座/出票结果为准（后端会按实际分配座位计价）。</div>
          </div>
          <div style="margin-top:10px;">
            <div class="muted small">乘车人与席别</div>
            ${passengerSummary || `<div class="muted small" style="margin-top:8px;">暂无乘客</div>`}
          </div>
          <div class="actions">
            <button class="btn btn-secondary" data-click="buyPrev" ${m.creating ? "disabled" : ""}>上一步</button>
            <button class="btn btn-primary" data-click="createOrder" ${m.creating ? "disabled" : ""}>提交订单并锁座</button>
          </div>
        `
        : `
          <div class="muted small" style="margin-top:6px;">先选择乘客与席别，下一步确认订单后再提交锁座。</div>
          ${ps}
          <div class="actions">
            <button class="btn btn-secondary" data-click="addPassenger">添加乘客</button>
            <button class="btn btn-primary" data-click="buyNext">下一步</button>
          </div>
        `;

      return `
        <div class="modal-mask" data-click="closeModal">
          <div class="modal" onclick="event.stopPropagation()">
            <div class="modal-head">
              <div><strong>预订</strong> · ${htmlesc(m.train.departure_station)} → ${htmlesc(m.train.arrival_station)}</div>
              <button class="close" data-click="closeModal">关闭</button>
            </div>
            <div class="modal-body">
              ${stepper}
              <div class="muted small">车次：${htmlesc(m.train.train_id)} · 运行时长：${formatMin(m.train.runtime_minutes)}</div>
              ${seatDetailHint}
              ${shortageTypes.length ? `<div class="banner warn" style="margin-top:10px;">余票提示：${htmlesc(shortageTypes.join("；"))}</div>` : ""}
              ${body}
            </div>
          </div>
        </div>
      `;
    }

    if (m.type === "pay") {
      const payUrl = m.payResp?.pay_url || "";
      if (!m.payChannel) m.payChannel = "ALIPAY";
      const statusText = m.orderInfo?.order_status ? `订单状态：${m.orderInfo.order_status}` : (m.payResp?.order_status ? `订单状态：${m.payResp.order_status}` : "");
      return `
        <div class="modal-mask" data-click="closeModal">
          <div class="modal" onclick="event.stopPropagation()">
            <div class="modal-head">
              <div><strong>支付</strong> · 订单 ${htmlesc(m.order.order_id)}</div>
              <button class="close" data-click="closeModal">关闭</button>
            </div>
            <div class="modal-body">
              <div class="stepper">
                <div class="step done">1 乘车人</div>
                <div class="step done">2 确认订单</div>
                <div class="step active">3 支付出票</div>
              </div>
              <div class="card" style="padding:12px;">
                <div class="row">
                  <div>
                    <div class="muted small">区间</div>
                    <div>${htmlesc(m.train.departure_station)} → ${htmlesc(m.train.arrival_station)}</div>
                  </div>
                  <div>
                    <div class="muted small">支付截止</div>
                    <div>${new Date((m.order.pay_deadline_unix || 0) * 1000).toLocaleString()}</div>
                  </div>
                </div>
              </div>
              <div style="margin-top:10px;" class="muted small">${htmlesc(statusText)}</div>
              <div style="margin-top:10px;display:flex;gap:10px;align-items:center;">
                <div class="muted small">支付渠道</div>
                <select class="select" style="width:160px;padding:8px 10px;" data-click="noop" data-pay-channel="1">
                  <option value="ALIPAY" ${m.payChannel === "ALIPAY" ? "selected" : ""}>支付宝</option>
                  <option value="WECHAT" ${m.payChannel === "WECHAT" ? "selected" : ""}>微信支付</option>
                  <option value="CARD" ${m.payChannel === "CARD" ? "selected" : ""}>银行卡</option>
                </select>
              </div>
              <div class="actions">
                <button class="btn btn-primary" data-click="payOrder" ${m.paying ? "disabled" : ""}>生成支付链接</button>
                <button class="btn btn-light" data-click="openPayURL" ${payUrl ? "" : "disabled"}>打开支付页</button>
                <button class="btn btn-light" data-click="mockPay" ${m.payResp?.pay_no ? "" : "disabled"}>模拟回调成功</button>
                <button class="btn btn-light" data-click="refreshPayStatus">刷新状态</button>
              </div>
              ${payUrl ? `<div class="card" style="padding:12px;margin-top:12px;word-break:break-all;"><div class="muted small">pay_url</div><div>${htmlesc(payUrl)}</div></div>` : ""}
            </div>
          </div>
        </div>
      `;
    }

    if (m.type === "change") {
      const loading = Boolean(m.loading);
      const submitting = Boolean(m.submitting);

      const rows = (m.trains || []).map((t) => {
        const active = m.selected?.train_id === t.train_id;
        const disabled = Number(t.remaining_seat_count || 0) <= 0;
        return `
          <div class="train-item" style="grid-template-columns: 1.4fr 1fr 1fr 120px; border-color:${active ? "rgba(22,119,255,0.45)" : "var(--border)"}; opacity:${disabled ? "0.65" : "1"}">
            <div class="train-main">
              <div class="train-title">${htmlesc(t.train_id)} <span class="tag">${htmlesc(t.train_type)}</span></div>
              <div class="train-sub">${htmlesc(t.departure_station)} → ${htmlesc(t.arrival_station)} · ${formatMin(t.runtime_minutes)}</div>
            </div>
            <div>
              <div class="muted small">席别</div>
              <div>${htmlesc(t.seat_type || "-")}</div>
            </div>
            <div>
              <div class="price">¥${Number(t.seat_price || 0).toFixed(2)}</div>
              <div class="remain ${t.remaining_seat_count > 0 ? "ok" : "bad"}">${t.remaining_seat_count > 0 ? `余票 ${t.remaining_seat_count}` : "无票"}</div>
            </div>
            <div style="display:flex;justify-content:flex-end;">
              <button class="btn btn-secondary" ${disabled ? "disabled" : ""} title="${disabled ? "无票不可选" : "选择该车次"}" data-click="pickChangeTrain" data-train-id="${htmlesc(t.train_id)}">选择</button>
            </div>
          </div>
        `;
      }).join("");

      const skeletonRows = Array.from({ length: 4 }).map(() => {
        return `
          <div class="train-item" style="border-style:dashed;">
            <div class="train-main">
              <div class="skeleton skeleton-line lg" style="width: 220px;"></div>
              <div class="skeleton skeleton-line sm" style="width: 280px; margin-top: 10px;"></div>
            </div>
            <div>
              <div class="skeleton skeleton-line sm" style="width: 64px;"></div>
              <div class="skeleton skeleton-line" style="width: 120px; margin-top: 10px;"></div>
            </div>
            <div>
              <div class="skeleton skeleton-line lg" style="width: 120px;"></div>
              <div class="skeleton skeleton-line sm" style="width: 140px; margin-top: 10px;"></div>
            </div>
            <div style="display:flex;justify-content:flex-end;">
              <div class="skeleton" style="height: 40px; width: 90px;"></div>
            </div>
          </div>
        `;
      }).join("");

      const listArea = loading
        ? `<div style="margin-top:10px;">${skeletonRows}</div>`
        : (rows || `<div class="empty" style="margin-top:10px;"><h3>没有可改签的车次</h3><p>试试调整日期或区间后重新查询。</p></div>`);

      return `
        <div class="modal-mask" data-click="closeModal">
          <div class="modal" onclick="event.stopPropagation()">
            <div class="modal-head">
              <div><strong>改签</strong> · 原订单 ${htmlesc(m.order_id)}</div>
              <button class="close" data-click="closeModal">关闭</button>
            </div>
            <div class="modal-body">
              <div class="modal-grid">
                <div>
                  <div class="row">
                    <div class="field">
                      <label>出发站</label>
                      <input class="input" value="${htmlesc(m.departure_station)}" data-field="departure_station" ${submitting ? "disabled" : ""} />
                    </div>
                    <div class="field">
                      <label>到达站</label>
                      <input class="input" value="${htmlesc(m.arrival_station)}" data-field="arrival_station" ${submitting ? "disabled" : ""} />
                    </div>
                  </div>
                  <div class="row" style="margin-top:10px;">
                    <div class="field">
                      <label>出行日期（用于查询）</label>
                      <input class="input" type="date" value="${htmlesc(m.travel_date)}" data-field="travel_date" ${submitting ? "disabled" : ""} />
                    </div>
                    <div style="display:flex;align-items:end;justify-content:flex-end;">
                      <button class="btn btn-primary" data-click="changeSearch" ${loading || submitting ? "disabled" : ""}>查询可改签车次</button>
                    </div>
                  </div>

                  <div class="banner info" style="margin-top:10px;">选择一个新车次后点击“提交改签”，会调用 /api/v1/order/change。</div>
                  ${listArea}

                  <div class="actions">
                    <button class="btn btn-secondary" data-click="closeModal" ${submitting ? "disabled" : ""}>取消</button>
                    <button class="btn btn-primary" data-click="submitChange" ${submitting || !m.selected ? "disabled" : ""}>提交改签</button>
                  </div>
                </div>

                <div class="sticky">
                  <div class="card" style="padding:12px; box-shadow:none;">
                    <div class="muted small">已选新车次</div>
                    ${m.selected ? `
                      <div style="margin-top:6px;"><strong>${htmlesc(m.selected.train_id)}</strong> <span class="tag">${htmlesc(m.selected.train_type)}</span></div>
                      <div class="muted small" style="margin-top:6px;">${htmlesc(m.selected.departure_station)} → ${htmlesc(m.selected.arrival_station)} · ${formatMin(m.selected.runtime_minutes)}</div>
                      <div style="margin-top:8px;"><span class="price" style="font-size:16px;">¥${Number(m.selected.seat_price || 0).toFixed(2)}</span> <span class="muted small">余票 ${Number(m.selected.remaining_seat_count || 0)}</span></div>
                    ` : `<div style="margin-top:8px;" class="muted small">未选择</div>`}
                  </div>

                  ${m.resp ? `
                    <div class="card" style="padding:12px;margin-top:12px; box-shadow:none;">
                      <div class="muted small">改签结果</div>
                      <div style="margin-top:6px;">新订单：<a class="btn btn-link" href="#/order/${htmlesc(m.resp.new_order_id || "")}">${htmlesc(m.resp.new_order_id || "-")}</a></div>
                      <div class="muted small" style="margin-top:6px;">差额：${Number(m.resp.refund_diff_amount || 0).toFixed(2)}（>0 退，<0 补）</div>
                    </div>
                  ` : ""}
                </div>
              </div>
            </div>
          </div>
        </div>
      `;
    }

    return "";
  }

  function wireModalInputs() {
    const m = state.modal;
    if (!m || m.type !== "buy") return;
    document.querySelectorAll(".modal [data-field]").forEach((el) => {
      const idx = Number(el.getAttribute("data-idx"));
      const field = el.getAttribute("data-field");
      if (el.tagName === "SELECT") el.value = m.passengers[idx][field] || "";
      el.addEventListener("input", (ev) => actions.updatePassenger(ev, el));
      el.addEventListener("change", (ev) => actions.updatePassenger(ev, el));
    });
  }

  function wirePayInputs() {
    const m = state.modal;
    if (!m || m.type !== "pay") return;
    const sel = document.querySelector(".modal [data-pay-channel]");
    if (sel) {
      sel.value = m.payChannel || "ALIPAY";
      sel.addEventListener("change", () => actions.updatePayChannel(null, sel));
    }
  }

  function wireVerifyInputs() {
    const m = state.modal;
    if (!m || m.type !== "verify_realname") return;
    const rn = document.getElementById("verify_real_name");
    const idc = document.getElementById("verify_id_card");
    const ph = document.getElementById("verify_phone");
    if (rn) rn.addEventListener("input", () => { m.real_name = rn.value; });
    if (idc) idc.addEventListener("input", () => { m.id_card = idc.value; });
    if (ph) ph.addEventListener("input", () => { m.phone = ph.value; });
  }

  function wireChangeInputs() {
    const m = state.modal;
    if (!m || m.type !== "change") return;
    document.querySelectorAll(".modal [data-field]").forEach((el) => {
      el.addEventListener("input", (ev) => actions.updateChangeField(ev, el));
      el.addEventListener("change", (ev) => actions.updateChangeField(ev, el));
    });
  }

  function viewChat() {
    if (!state.chat.open) {
      return `<div class="chat-fab" data-click="toggleChat">💬</div>`;
    }
    const msgs = state.chat.msgs.map((m) => {
      const isUser = m.role === "user";
      const toolView = m.tool ? `
        <div class="chat-tool">
          <div class="chat-tool-head">
            <span>🛠 调用工具：${htmlesc(m.tool.name)}</span>
            <span>${m.tool.error ? `<span style="color:var(--bad)">失败</span>` : `<span style="color:var(--ok)">成功</span>`}</span>
          </div>
          <div class="chat-tool-content">${htmlesc(m.tool.args)}</div>
          ${m.tool.result ? `<div style="margin-top:6px;border-top:1px dashed #e5e7eb;padding-top:6px;" class="chat-tool-content">${htmlesc(m.tool.result.slice(0, 120) + (m.tool.result.length > 120 ? "..." : ""))}</div>` : ""}
        </div>
      ` : "";
      return `
        <div class="chat-msg ${isUser ? "user" : "ai"}">
          <div class="chat-bubble">${htmlesc(m.content)}</div>
          ${toolView}
        </div>
      `;
    }).join("");

    return `
      <div class="chat-fab" data-click="toggleChat" style="transform: scale(0); opacity: 0;"></div>
      <div class="chat-window">
        <div class="chat-header">
          <div style="font-weight:700;display:flex;align-items:center;gap:8px;">
            <span>🤖 智能助手</span>
            <span class="tag" style="font-weight:400;font-size:10px;">Beta</span>
          </div>
          <button class="close" data-click="toggleChat">×</button>
        </div>
        <div id="chat_body" class="chat-body">
          ${msgs}
          ${state.chat.loading ? `<div class="chat-msg ai"><div class="chat-bubble"><div class="muted small">思考中...</div></div></div>` : ""}
        </div>
        <div class="chat-footer">
          <div class="chat-input-box">
            <textarea id="chat_input" class="chat-input" placeholder="输入你想做的事..." rows="1" data-input="chat" data-keydown="chatKeydown"></textarea>
            <button class="btn btn-primary" data-click="sendChat" ${state.chat.loading ? "disabled" : ""}>发送</button>
          </div>
        </div>
      </div>
    `;
  }

  function render() {
    const active = document.activeElement;
    const focusSnapshot = (() => {
      if (!active) return null;
      if (!$app.contains(active)) return null;
      const id = active.id;
      if (!id) return null;
      const tag = active.tagName;
      const isText = tag === "INPUT" || tag === "TEXTAREA";
      const shouldKeep = active.closest?.(".modal") || id === "dep_input" || id === "arr_input";
      if (!shouldKeep) return null;
      let selectionStart = null;
      let selectionEnd = null;
      if (isText && typeof active.selectionStart === "number" && typeof active.selectionEnd === "number") {
        selectionStart = active.selectionStart;
        selectionEnd = active.selectionEnd;
      }
      return { id, tag, selectionStart, selectionEnd };
    })();

    const { raw: hash, path, q } = parseHash();
    let view = "";
    if (path === "#/" || path.startsWith("#/search")) view = viewHome();
    else if (path === "#/trains" || path === "#/order/confirm") view = viewSearch();
    else if (path === "#/pay") view = viewPay();
    else if (path === "#/login") view = viewLogin();
    else if (path === "#/register") view = viewRegister();
    else if (path === "#/orders") view = viewOrders();
    else if (path === "#/account" || path === "#/me") view = viewMe();
    else if (path === "#/kb") view = viewKB();
    else if (path.startsWith("#/order/")) view = viewOrderDetail(path.slice("#/order/".length));
    else view = viewHome();

    const toast = state.toast ? `<div class="toast ${toastClassName(state.toast)}">${htmlesc(state.toast.msg)}</div>` : "";
    $app.innerHTML = view + viewModal() + viewChat() + toast;
    mountEvents();
    wireModalInputs();
    wirePayInputs();
    wireVerifyInputs();
    wireChangeInputs();
    wireSuggestInputs();
    wireVirtualList();
    syncRemainWS();

    const chatIn = document.getElementById("chat_input");
    if (chatIn) {
      chatIn.addEventListener("input", (ev) => actions.updateChatInput(ev, chatIn));
      chatIn.addEventListener("keydown", (ev) => actions.chatKeydown(ev, chatIn));
      chatIn.value = state.chat.input || "";
    }

    const kbSearchIn = document.getElementById("kb_query_input");
    if (kbSearchIn) {
      kbSearchIn.addEventListener("input", (ev) => actions.updateKBSearch(ev, kbSearchIn));
      kbSearchIn.addEventListener("keydown", (ev) => {
        if (ev.key === "Enter") actions.kbSearch();
      });
    }

    document.querySelectorAll('[data-field]').forEach(el => {
      el.addEventListener("input", (ev) => actions.updateKBField(ev, el));
    });

    if (focusSnapshot) {
      const el = document.getElementById(focusSnapshot.id);
      if (el && el.tagName === focusSnapshot.tag) {
        try {
          el.focus({ preventScroll: true });
          if ((el.tagName === "INPUT" || el.tagName === "TEXTAREA") && typeof el.setSelectionRange === "function" && focusSnapshot.selectionStart !== null) {
            el.setSelectionRange(focusSnapshot.selectionStart, focusSnapshot.selectionEnd ?? focusSnapshot.selectionStart);
          }
        } catch {}
      }
    }
  }

  window.addEventListener("hashchange", () => {
    const { path, q } = parseHash();

    if (path !== "#/pay") stopPayPolling();

    if (path === "#/trains") actions.resetSearch();
    if (path === "#/orders") loadOrders();
    if (path.startsWith("#/order/") && path !== "#/order/confirm") loadOrderDetail(path.slice("#/order/".length));
    if (path === "#/account" || path === "#/me") {
      loadProfile();
      loadPassengers();
      if (String(q.get("verify") || "") === "1") {
        (async () => {
          const prof = state.profile.data?.user_info ? state.profile.data : await fetchProfileSilently();
          if (!isRealNameVerified(prof)) actions.openVerify();
        })();
      }
    }
    if (path === "#/pay") {
      const orderId = q.get("order_id") || state.payDraft?.order?.order_id || "";
      if (orderId) {
        startPayPolling(orderId);
        actions.payPageRefresh();
      }
    }
    if (path === "#/order/confirm") {
      const trainId = q.get("train_id");
      if (trainId) {
        (async () => {
          const ok = await ensureBuyRuleVerified({ next: "#/order/confirm?train_id=" + encodeURIComponent(trainId) });
          if (!ok) return;
          const item = state.trains.find((t) => t.train_id === trainId);
          if (item) openBuyModal(item);
          else setToast("请先在车次列表选择要预订的车次");
        })();
      }
    }
    if (path === "#/kb") {
      // 首次进入 KB 页不自动查，等用户搜
    }
    render();
  });

  if (!location.hash) setHash("#/");
  const init = parseHash();
  if (init.path === "#/trains") actions.resetSearch();
  if (init.path === "#/orders") loadOrders();
  if (init.path.startsWith("#/order/") && init.path !== "#/order/confirm") loadOrderDetail(init.path.slice("#/order/".length));
  if (init.path === "#/account" || init.path === "#/me") {
    loadProfile();
    loadPassengers();
    if (String(init.q.get("verify") || "") === "1") {
      (async () => {
        const prof = state.profile.data?.user_info ? state.profile.data : await fetchProfileSilently();
        if (!isRealNameVerified(prof)) actions.openVerify();
      })();
    }
  }
  if (init.path === "#/pay") {
    const orderId = init.q.get("order_id") || state.payDraft?.order?.order_id || "";
    if (orderId) startPayPolling(orderId);
  }
  if (init.path === "#/order/confirm") {
    const trainId = init.q.get("train_id");
    if (trainId) {
      (async () => {
        const ok = await ensureBuyRuleVerified({ next: "#/order/confirm?train_id=" + encodeURIComponent(trainId) });
        if (!ok) return;
        const item = state.trains.find((t) => t.train_id === trainId);
        if (item) openBuyModal(item);
        else setToast("请先在车次列表选择要预订的车次");
      })();
    }
  }
  if (state.me.token) {
    fetchProfileSilently();
    fetchPassengersSilently();
  }
  render();
})();
