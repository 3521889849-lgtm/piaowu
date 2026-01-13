package main

import (
	"context" // 必须导入，Test方法要用到ctx参数
	_ "example_shop/common/init"
	"example_shop/kitex_gen/coupon"
	"example_shop/kitex_gen/coupon/couponservice"

	"log"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
)

// 空结构体（无任何业务逻辑）
type CouponServiceImpl struct{}

// 【关键】补全Test方法的空实现（解决爆红核心）
// 方法签名必须和Kitex生成的接口完全一致，一字不差！
func (s *CouponServiceImpl) Test(ctx context.Context, req *coupon.EmptyReq) (*coupon.BaseResp, error) {
	// 纯空逻辑：啥都不做，只返回默认成功响应
	return &coupon.BaseResp{
		Code: 200,
		Msg:  "空方法执行成功（无任何业务逻辑）",
	}, nil
}

func main() {
	// 启动空服务
	svr := couponservice.NewServer(
		new(CouponServiceImpl), // 现在这个结构体已实现全部接口，不再爆红
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "coupon_service",
		}),
	)

	log.Println("✅ 极简空服务启动成功！")
	if err := svr.Run(); err != nil {
		log.Println("启动失败：", err)
	}
}
