<template>
	<view class="container">
		<view class="card" v-if="detail">
			<view class="row">
				<text class="label">审核ID</text>
				<text>{{ detail.task_info.audit_id }}</text>
			</view>
			<view class="row">
				<text class="label">提交人</text>
				<text>{{ detail.task_info.submitter_id }}</text>
			</view>
			<view class="row">
				<text class="label">状态</text>
				<text class="status">{{ getStatus(detail.task_info.status) }}</text>
			</view>
			<view class="row column">
				<text class="label">申请内容</text>
				<text class="content-box">{{ detail.task_info.content }}</text>
			</view>
		</view>

		<view class="card" v-if="detail && getDecision(detail.task_info.extra)">
			<view class="section-title">决策分析</view>
			<view class="decision-info">
				<view class="decision-row">
					<text class="decision-label">系统建议</text>
					<text :class="['decision-val', getDecision(detail.task_info.extra).final_action]">{{ translateAction(getDecision(detail.task_info.extra).final_action) }}</text>
				</view>
				<view class="decision-row">
					<text class="decision-label">模型评分</text>
					<text>{{ (getDecision(detail.task_info.extra).model_score * 100).toFixed(1) }}%</text>
				</view>
				<view class="decision-row">
					<text class="decision-label">动态阈值</text>
					<text>{{ getDecision(detail.task_info.extra).dynamic_threshold }}</text>
				</view>
				<view class="decision-row" v-if="getDecision(detail.task_info.extra).rule_matched">
					<text class="decision-label">命中规则</text>
					<text class="rule-tag">是</text>
				</view>
			</view>
		</view>
		
		<view class="card" v-if="detail && detail.logs && detail.logs.length > 0">
			<view class="section-title">操作日志</view>
			<view class="log-item" v-for="(log, index) in detail.logs" :key="index">
				<view class="log-header">
					<text class="operator">{{ log.operator_id }}</text>
					<text class="action">{{ log.operation }}</text>
				</view>
				<view class="log-time">{{ formatTime(log.create_time) }}</view>
				<view class="log-remark" v-if="log.remark">备注: {{ log.remark }}</view>
			</view>
		</view>
		
		<view class="footer-bar" v-if="detail && detail.task_info.status === 1">
			<button class="btn-action reject" @click="openAuditModal(false)">驳回</button>
			<button class="btn-action pass" @click="openAuditModal(true)">通过</button>
		</view>
		
		<!-- 简单的模态框模拟 -->
		<view class="modal-mask" v-if="showModal">
			<view class="modal">
				<view class="modal-title">{{ isPass ? '通过审核' : '驳回审核' }}</view>
				<textarea class="modal-input" v-model="remark" placeholder="请输入审核意见..."></textarea>
				<view class="modal-btns">
					<button class="modal-btn cancel" @click="showModal = false">取消</button>
					<button class="modal-btn confirm" @click="submitAudit">确定</button>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
	import { request, AuditStatusMap } from '../../common/api.js';
	
	export default {
		data() {
			return {
				auditId: 0,
				detail: null,
				showModal: false,
				isPass: false,
				remark: ''
			}
		},
		onLoad(options) {
			if (options.id) {
				this.auditId = options.id;
				this.loadDetail();
			}
		},
		methods: {
			translateAction(action) {
				const map = {
					'Pass': '通过',
					'Reject': '拒绝',
					'Review': '人工审核'
				};
				return map[action] || action;
			},
			getStatus(status) {
				return AuditStatusMap[status] || status;
			},
			formatTime(time) {
				return time ? time.replace('T', ' ').substring(0, 19) : '';
			},
			getDecision(extra) {
				if (!extra) return null;
				try {
					return JSON.parse(extra);
				} catch (e) {
					return null;
				}
			},
			async loadDetail() {
				try {
					const res = await request({
						url: `/record?audit_id=${this.auditId}`,
						method: 'GET'
					});
					if (res.base_resp && res.base_resp.code === 0) {
						this.detail = res.detail;
					}
				} catch (e) {
					console.error(e);
				}
			},
			openAuditModal(pass) {
				this.isPass = pass;
				this.remark = pass ? '同意申请' : '信息不符，予以驳回';
				this.showModal = true;
			},
			async submitAudit() {
				try {
					const res = await request({
						url: '/manual/process',
						method: 'POST',
						data: {
							audit_id: Number(this.auditId),
							auditor_id: '999',
							is_passed: this.isPass,
							remark: this.remark,
							opinion: this.remark
						}
					});
					
					if (res.base_resp && res.base_resp.code === 0) {
						uni.showToast({ title: '操作成功' });
						this.showModal = false;
						this.loadDetail();
					} else {
						uni.showToast({ title: res.base_resp.msg, icon: 'none' });
					}
				} catch (e) {
					console.error(e);
				}
			}
		}
	}
</script>

<style>
	.container {
		padding-bottom: 120rpx;
	}
	.row {
		display: flex;
		justify-content: space-between;
		padding: 10rpx 0;
		border-bottom: 1rpx solid #eee;
	}
	.row.column {
		flex-direction: column;
	}
	.label {
		color: #999;
		margin-right: 20rpx;
	}
	.content-box {
		background-color: #f8f8f8;
		padding: 10rpx;
		margin-top: 10rpx;
		border-radius: 4rpx;
		word-break: break-all;
	}
	.status {
		color: #007aff;
		font-weight: bold;
	}
	.section-title {
		font-size: 30rpx;
		font-weight: bold;
		margin-bottom: 20rpx;
		border-left: 6rpx solid #007aff;
		padding-left: 16rpx;
	}
	.log-item {
		padding: 16rpx 0;
		border-bottom: 1rpx solid #f0f0f0;
	}
	.log-header {
		display: flex;
		justify-content: space-between;
	}
	.action {
		font-weight: bold;
	}
	.log-time {
		font-size: 24rpx;
		color: #999;
		margin: 4rpx 0;
	}
	.footer-bar {
		position: fixed;
		bottom: 0;
		left: 0;
		width: 100%;
		background-color: #fff;
		display: flex;
		padding: 20rpx;
		box-shadow: 0 -2rpx 10rpx rgba(0,0,0,0.05);
		box-sizing: border-box;
	}
	.btn-action {
		flex: 1;
		margin: 0 10rpx;
		color: #fff;
		font-size: 28rpx;
	}
	.pass { background-color: #07c160; }
	.reject { background-color: #fa5151; }
	
	.decision-info {
		padding: 10rpx 0;
	}
	.decision-row {
		display: flex;
		justify-content: space-between;
		padding: 6rpx 0;
		font-size: 26rpx;
	}
	.decision-label {
		color: #666;
	}
	.decision-val {
		font-weight: bold;
	}
	.decision-val.Pass { color: #07c160; }
	.decision-val.Reject { color: #fa5151; }
	.decision-val.Review { color: #ff9900; }
	.rule-tag {
		background-color: #e6f7ff;
		color: #1890ff;
		padding: 2rpx 12rpx;
		border-radius: 4rpx;
		font-size: 22rpx;
	}
	
	.modal-mask {
		position: fixed;
		top: 0; left: 0; right: 0; bottom: 0;
		background-color: rgba(0,0,0,0.5);
		display: flex;
		justify-content: center;
		align-items: center;
		z-index: 999;
	}
	.modal {
		background-color: #fff;
		width: 80%;
		border-radius: 12rpx;
		padding: 30rpx;
	}
	.modal-title {
		font-size: 32rpx;
		font-weight: bold;
		text-align: center;
		margin-bottom: 30rpx;
	}
	.modal-input {
		width: 100%;
		height: 160rpx;
		background-color: #f8f8f8;
		padding: 10rpx;
		margin-bottom: 30rpx;
		box-sizing: border-box;
	}
	.modal-btns {
		display: flex;
	}
	.modal-btn {
		flex: 1;
		margin: 0 10rpx;
		font-size: 28rpx;
	}
	.confirm { background-color: #007aff; color: #fff; }
</style>
