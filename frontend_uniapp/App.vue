<script>
	export default {
		onLaunch: function() {
			console.log('App Launch')
		},
		onShow: function() {
			console.log('App Show')
		},
		onHide: function() {
			console.log('App Hide')
		}
	}
</script>

<style>
	/*每个页面公共css */
	page {
		background-color: #f5f5f5;
		font-size: 28rpx;
		color: #333;
	}
	.container {
		padding: 20rpx;
	}
	.btn {
		background-color: #007aff;
		color: #fff;
		border-radius: 8rpx;
		padding: 20rpx;
		text-align: center;
		margin: 20rpx 0;
	}
	.btn:active {
		opacity: 0.8;
	}
	.card {
		background-color: #fff;
		border-radius: 12rpx;
		padding: 20rpx;
		margin-bottom: 20rpx;
		box-shadow: 0 2rpx 10rpx rgba(0,0,0,0.05);
	}
</style>
