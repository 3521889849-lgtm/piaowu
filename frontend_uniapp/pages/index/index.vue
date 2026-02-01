<template>
	<view class="container">
		<view class="header">
			<text class="title">待处理任务</text>
			<text class="refresh" @click="loadData">刷新</text>
		</view>
		
		<view class="list">
			<view class="card" v-for="(item, index) in list" :key="index" @click="goDetail(item)">
				<view class="card-header">
					<text class="biz-type">{{ getBizType(item.biz_type) }}</text>
					<text class="status">{{ getStatus(item.status) }}</text>
				</view>
				<view class="card-content">
					<text>ID: {{ item.audit_id }}</text>
					<text class="desc">{{ item.content }}</text>
					<text class="time">{{ formatTime(item.apply_time) }}</text>
				</view>
			</view>
			<view v-if="list.length === 0" class="empty">暂无待审核任务</view>
		</view>
	</view>
</template>

<script>
	import { request, BizTypeMap, AuditStatusMap } from '../../common/api.js';
	
	export default {
		data() {
			return {
				list: [],
				page: 1,
				loading: false
			}
		},
		onShow() {
			this.loadData();
		},
		onPullDownRefresh() {
			this.loadData();
		},
		methods: {
			getBizType(type) {
				return BizTypeMap[type] || '未知';
			},
			getStatus(status) {
				return AuditStatusMap[status] || '未知';
			},
			formatTime(time) {
				return time ? time.replace('T', ' ').substring(0, 19) : '';
			},
			async loadData() {
				this.loading = true;
				try {
					const res = await request({
						url: '/manual/tasks',
						method: 'POST',
						data: {
							status: 1, // Pending
							pagination: {
								page_num: 1,
								page_size: 20
							}
						}
					});
					
					if (res.base_resp && res.base_resp.code === 0) {
						this.list = res.tasks || [];
					} else {
						uni.showToast({ title: res.base_resp.msg, icon: 'none' });
					}
				} catch (e) {
					console.error(e);
				} finally {
					this.loading = false;
					uni.stopPullDownRefresh();
				}
			},
			goDetail(item) {
				uni.navigateTo({
					url: `/pages/detail/detail?id=${item.audit_id}`
				});
			}
		}
	}
</script>

<style>
	.header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 20rpx 0;
	}
	.title {
		font-size: 32rpx;
		font-weight: bold;
	}
	.refresh {
		color: #007aff;
	}
	.card-header {
		display: flex;
		justify-content: space-between;
		margin-bottom: 10rpx;
	}
	.biz-type {
		background-color: #e8f4ff;
		color: #007aff;
		padding: 4rpx 12rpx;
		border-radius: 4rpx;
		font-size: 24rpx;
	}
	.status {
		color: #ff9900;
	}
	.card-content {
		display: flex;
		flex-direction: column;
		gap: 8rpx;
	}
	.desc {
		color: #666;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.time {
		color: #999;
		font-size: 24rpx;
		align-self: flex-end;
	}
	.empty {
		text-align: center;
		color: #999;
		padding: 100rpx 0;
	}
</style>
