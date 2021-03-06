package menu

import (
	"fmt"

	"github.com/micro-plat/hydra/component"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/sso/modules/const/sql"
)

type IMenu interface {
	Query(uid int64, sysid int) ([]map[string]interface{}, error)
	Verify(uid int64, sysid int, menuURL string) error
}

type Menu struct {
	c component.IContainer
}

func NewMenu(c component.IContainer) *Menu {
	return &Menu{
		c: c,
	}
}

//Query 获取用户指定系统的菜单信息
func (l *Menu) Query(uid int64, sysid int) ([]map[string]interface{}, error) {
	db := l.c.GetRegularDB()
	data, _, _, err := db.Query(sql.QueryUserMenus, map[string]interface{}{
		"user_id": uid,
		"sys_id":  sysid,
	})
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, 4)
	for _, row1 := range data {
		if row1.GetInt("parent") == 0 && row1.GetInt("level_id") == 1 {
			children1 := make([]map[string]interface{}, 0, 4)
			for _, row2 := range data {
				if row2.GetInt("parent") == row1.GetInt("id") && row2.GetInt("level_id") == 2 {
					children2 := make([]map[string]interface{}, 0, 8)
					for _, row3 := range data {
						if row3.GetInt("parent") == row2.GetInt("id") && row3.GetInt("level_id") == 3 {
							children2 = append(children2, row3)
						}
					}
					children1 = append(children1, row2)
					row2["children"] = children2
				}
			}
			row1["children"] = children1
			result = append(result, row1)
		}
	}

	return result, nil
}

//Verify 获取用户指定系统的菜单信息
func (l *Menu) Verify(uid int64, sysid int, menuURL string) error {
	db := l.c.GetRegularDB()
	//根据用户名密码，查询用户信息
	data, _, _, err := db.Scalar(sql.QueryUserMenu, map[string]interface{}{
		"user_id": uid,
		"sys_id":  sysid,
		"path":    menuURL,
	})
	if err != nil {
		return err
	}
	if fmt.Sprint(data) == "1" {
		return nil
	}
	return context.NewError(context.ERR_FORBIDDEN, fmt.Errorf("未查找到菜单"))
}
