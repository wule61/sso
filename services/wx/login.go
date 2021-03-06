package wx

import (
	"encoding/json"
	"fmt"

	"github.com/micro-plat/hydra/component"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/lib4go/db"
	"github.com/micro-plat/lib4go/net/http"
	"github.com/micro-plat/sso/modules/app"
	"github.com/micro-plat/sso/modules/member"
	"github.com/micro-plat/wechat/mp/oauth2"
)

type LoginHandler struct {
	c    component.IContainer
	m    member.IMember
	code member.ICodeMember
}

func NewLoginHandler(container component.IContainer) (u *LoginHandler) {
	return &LoginHandler{
		c:    container,
		m:    member.NewMember(container),
		code: member.NewCodeMember(container),
	}
}

func (u *LoginHandler) getLoginURL(sysid int) string {
	conf := app.GetConf(u.c)
	url := fmt.Sprintf("%s?sysid=%d", conf.WXLoginURL, sysid)
	return oauth2.AuthCodeURL(conf.AppID, url, "snsapi_base", "")
}

//Handle 使用微信code查询用户openid,并登录，推送到ws端code
func (u *LoginHandler) Handle(ctx *context.Context) (r interface{}) {
	if err := ctx.Request.Check("code", "sysid"); err != nil {
		return context.NewError(context.ERR_NOT_ACCEPTABLE, err)
	}
	ctx.Log.Info("1. 根据code查询用户openid")
	sysid := ctx.Request.GetInt("sysid", 0)
	code := ctx.Request.GetString("code")
	conf := app.GetConf(u.c)
	endpoint := oauth2.NewEndpoint(conf.AppID, conf.Secret)
	url := endpoint.ExchangeTokenURL(code)
	client := http.NewHTTPClient()
	content, status, err := client.Get(url)
	if err != nil || status != 200 {
		return fmt.Errorf("远程请求失败:%s(%v)%d", url, err, status)
	}
	userInfo := make(db.QueryRow)
	if err = json.Unmarshal([]byte(content), &userInfo); err != nil {
		return err
	}
	ctx.Log.Info("2. 根据openid登录")
	openid := userInfo.GetString("openid")
	member, err := u.m.LoginByOpenID(openid, sysid)
	if err != nil {
		return fmt.Errorf("登录失败:(%v)%s(%s)", err, openid, content)
	}
	redirectURL := ctx.Request.GetString("redirect_uri")
	if redirectURL == "" {
		redirectURL = member.IndexURL
	}
	loginCode, err := u.code.Save(member)
	if err != nil {
		return fmt.Errorf("保存用户登录code失败:%v", err)
	}
	//设置jwt数据
	ctx.Log.Info("3. 返回登录端code")
	ctx.Response.SetJWT(member)
	return map[string]interface{}{
		"code": loginCode,
	}
}
