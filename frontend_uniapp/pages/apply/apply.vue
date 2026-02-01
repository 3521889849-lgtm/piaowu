<template>
	<view class="container">
		<view class="card">
			<view class="form-item">
				<text class="label">业务类型</text>
				<picker @change="bindTypeChange" :value="typeIndex" :range="types" range-key="label">
					<view class="picker">
						{{ types[typeIndex].label }}
					</view>
				</picker>
			</view>
			
			<view class="form-item">
				<text class="label">申请内容 (JSON)</text>
				<textarea class="textarea" v-model="content" placeholder="请输入申请内容..."></textarea>
			</view>
			
			<button class="btn" @click="submit" :loading="loading">提交申请</button>
		</view>
		
		<view class="tips">
			<text>提示：选择业务类型后会自动填充测试数据模板。</text>
		</view>
	</view>
</template>

<script>
	import { request, BizTypeMap } from '../../common/api.js';
	
	export default {
		data() {
			return {
				types: [
					{ value: 1, label: '车票订单' },
					{ value: 2, label: '酒店订单' },
					{ value: 3, label: '商户入驻' }
				],
				typeIndex: 0,
				content: '',
				loading: false
			}
		},
		mounted() {
			this.updateTemplate();
		},
		methods: {
			bindTypeChange(e) {
				this.typeIndex = e.detail.value;
				this.updateTemplate();
			},
			updateTemplate() {
				const type = this.types[this.typeIndex].value;
				if (type === 1) {
					this.content = JSON.stringify({
						order_id: 12345,
						passenger_name: "张三",
						passenger_id_card: "110101199003071234",
						departure_station: "北京",
						arrival_station: "上海",
						departure_time: "2026-05-01 12:00:00",
						order_amount: 100.50
					}, null, 2);
				} else if (type === 3) {
					this.content = "新商户入驻申请：\n商户名：示例店铺\n联系人：李四";
				} else {
					this.content = "";
				}
			},
			async submit() {
				if (!this.content) {
					uni.showToast({ title: '请输入内容', icon: 'none' });
					return;
				}
				
				this.loading = true;
				try {
					const res = await request({
						url: '/apply',
						method: 'POST',
						data: {
							biz_type: this.types[this.typeIndex].value,
							biz_id: Date.now().toString(),
							submitter_id: '1002',
							content: this.content,
							attachments: [],
							extra: {}
						}
					});
					
					if (res.base_resp && res.base_resp.code === 0) {
						uni.showToast({ title: '提交成功 ID:' + res.audit_id });
						setTimeout(() => {
							uni.switchTab({ url: '/pages/index/index' });
						}, 1500);
					} else {
						uni.showToast({ title: res.base_resp.msg, icon: 'none' });
					}
				} catch (e) {
					console.error(e);
				} finally {
					this.loading = false;
				}
			}
		}
	}
</script>

<style>
	.form-item {
		margin-bottom: 30rpx;
	}
	.label {
		display: block;
		margin-bottom: 10rpx;
		font-weight: bold;
	}
	.picker {
		background-color: #f8f8f8;
		padding: 20rpx;
		border-radius: 8rpx;
	}
	.textarea {
		width: 100%;
		height: 300rpx;
		background-color: #f8f8f8;
		padding: 20rpx;
		border-radius: 8rpx;
		box-sizing: border-box;
	}
	.tips {
		padding: 20rpx;
		color: #999;
		font-size: 24rpx;
	}
</style>
