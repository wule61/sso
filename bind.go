package main

import (
	"github.com/micro-plat/hydra/component"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/hydra/hydra"
	"github.com/micro-plat/sso/modules/app"
	mem "github.com/micro-plat/sso/modules/member"
	"github.com/micro-plat/sso/services/base"
	"github.com/micro-plat/sso/services/member"
	"github.com/micro-plat/sso/services/menu"
	"github.com/micro-plat/sso/services/qrcode"
	"github.com/micro-plat/sso/services/system"
	"github.com/micro-plat/sso/services/user"
	"github.com/micro-plat/sso/services/wx"
)

//bindConf 绑定启动配置， 启动时检查注册中心配置是否存在，不存在则引导用户输入配置参数并自动创建到注册中心
func bindConf(app *hydra.MicroApp) {
	app.Conf.API.SetMainConf(`{"address":":9091"}`)

	app.Conf.API.SetSubConf("app", `
			{
				"qrlogin-check-url":"http://sso.100bm.cn/member/wxlogin",
				"wx-login-url":"http://sso.100bm.cn/member/wxlogin",
				"appid":"wx9e02ddcc88e13fd4",
				"secret":"45d25cb71f3bee254c2bc6fc0dc0caf1",
				"wechat-url":"http://59.151.30.153:9999/wx9e02ddcc88e13fd4/wechat/token/get"
			}			
			`)
	app.Conf.API.SetSubConf("header", `
				{
					"Access-Control-Allow-Origin": "*", 
					"Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,PATCH,OPTIONS", 
					"Access-Control-Allow-Headers": "__jwt__", 
					"Access-Control-Allow-Credentials": "true"
				}
			`)

	app.Conf.API.SetSubConf("auth", `
		{
			"jwt": {
				"exclude": ["/sso/login","/sso/wxcode/get","/sso/sys/get","/qrcode/login","/qrcode/login/put","/sso/login/code"],
				"expireAt": 36000,
				"mode": "HS512",
				"name": "__jwt__",
				"secret": "12345678"
			}
		}
		`)

	app.Conf.WS.SetSubConf("app", `
			{
				"qrlogin-check-url":"http://sso.100bm.cn/member/wxlogin",
				"wx-login-url":"http://sso.100bm.cn/member/wxlogin",
				"appid":"wx9e02ddcc88e13fd4",
				"secret":"45d25cb71f3bee254c2bc6fc0dc0caf1",
				"wechat-url":"http://59.151.30.153:9999/wx9e02ddcc88e13fd4/wechat/token/get"
			}			
			`)
	app.Conf.Plat.SetVarConf("db", "db", `{			
			"provider":"ora",
			"connString":"sso/123456@orcl136",
			"maxOpen":10,
			"maxIdle":1,
			"lifeTime":10		
	}`)

	app.Conf.Plat.SetVarConf("cache", "cache", `
		{
			"proto":"redis",
			"addrs":[
					"192.168.0.110:6379",
					"192.168.0.122:6379",
					"192.168.0.134:6379",
					"192.168.0.122:6380",
					"192.168.0.110:6380",
					"192.168.0.134:6380"
			],
			"dial_timeout":10,
			"read_timeout":10,
			"write_timeout":10,
			"pool_size":10
	}
		
		`)
	app.Conf.Plat.SetVarConf("cache", "abc", `
			{
				"name":"杨磊"
		}
			
			`)

}

//bind 检查应用程序配置文件，并根据配置初始化服务
func bind(r *hydra.MicroApp) {
	bindConf(r)

	//每个请求执行前执行
	r.Handling(func(ctx *context.Context) (rt interface{}) {
		if ctx.GetContainer().GetServerType() != "api" {
			return nil
		}
		jwt, err := ctx.Request.GetJWTConfig() //获取jwt配置
		if err != nil {
			return err
		}
		for _, u := range jwt.Exclude { //排除指定请求
			if u == ctx.Service {
				return nil
			}
		}
		//检查jwt配置，并使用member中提供的函数缓存login信息到context中
		var m mem.LoginState
		if err := ctx.Request.GetJWT(&m); err != nil {
			return context.NewError(context.ERR_FORBIDDEN, err)
		}
		return mem.Save(ctx, &m)
	})

	//初始化
	r.Initializing(func(c component.IContainer) error {
		var conf app.Conf
		if err := c.GetAppConf(&conf); err != nil {
			return err
		}
		app.SaveConf(c, &conf)
		if err := conf.Valid(); err != nil {
			return err
		}

		//检查db配置是否正确
		if _, err := c.GetDB(); err != nil {
			return err
		}

		//检查缓存配置是否正确
		if _, err := c.GetCache(); err != nil {
			return err
		}
		r.Micro("/sso/wxcode/get", member.NewWxcodeHandler(conf.AppID, conf.Secret, conf.WechatTSAddr)) //获取已发送的微信验证码
		return nil
	})
	r.Micro("/sso/login", member.NewLoginHandler)     //用户名密码登录
	r.Micro("/sso/sys/get", system.NewSystemHandler)  //根据系统编号获取系统信息
	r.Micro("/sso/menu/get", menu.NewMenuHandler)     //获取用户所在系统的菜单信息
	r.Micro("/sso/popular", menu.NewPopularHandler)   //获取用户所在系统的常用菜单
	r.Micro("/sso/login/code", member.NewCodeHandler) //根据用户登录code设置jwt信息

	r.WS("/qrcode/login", qrcode.NewLoginHandler)    //二维码登录（获取二维码登录地址,接收用户扫码后的消息推送）
	r.Micro("/qrcode/login", qrcode.NewLoginHandler) //二维码登录(调用二维码登录接口地址，推送到PC端登录消息)

	r.Micro("/wx/login", wx.NewLoginHandler) //微信端登录

	r.Micro("/sso/login/check", member.NewCheckHandler) //用户登录状态检查，检查用户jwt是否有效
	//r.Micro("/sso/member/get", member.NewGetHandler)     //获取用户信息（不包括角色信息）
	r.Micro("/sso/member/query", member.NewQueryHandler) //查询登录用户信息

	r.Micro("/sso/menu/verify", menu.NewVerifyHandler) //检查用户菜单权限

	r.Micro("/sso/user/index", user.NewUserHandler)
	r.Micro("/sso/base/userrole", base.NewBaseUserHandler)
}
