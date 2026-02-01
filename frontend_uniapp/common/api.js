const BASE_URL = '/api/audit';

export const request = (options) => {
	return new Promise((resolve, reject) => {
		uni.request({
			url: BASE_URL + options.url,
			method: options.method || 'GET',
			data: options.data || {},
			header: {
				'content-type': 'application/json'
			},
			success: (res) => {
				if (res.statusCode === 200) {
					const data = res.data;
					if (data && data.base_resp && data.base_resp.code !== 0) {
						uni.showToast({
							title: data.base_resp.msg || '操作失败',
							icon: 'none'
						});
						reject(data);
					} else {
						resolve(data);
					}
				} else {
					uni.showToast({
						title: '服务异常: ' + res.statusCode,
						icon: 'none'
					});
					reject(res);
				}
			},
			fail: (err) => {
				uni.showToast({
					title: '网络连接失败',
					icon: 'none'
				});
				reject(err);
			}
		});
	});
};

export const BizTypeMap = {
	1: '车票订单',
	2: '酒店订单',
	3: '商户入驻',
	4: '资源上传'
};

export const AuditStatusMap = {
	1: '待审核',
	2: '审核中',
	3: '通过',
	4: '驳回',
	5: '取消'
};
