package Wsy

import (
    "io"
    "errors"
    "strings"
	"net/http"
    "github.com/gin-gonic/gin"
)
type WsyGin struct {
    engine *gin.Engine
    Path   string
    Port   string
    Dirs   bool
	Debug  bool
}
//全局初始化GinValid struct  使用方法  Wsy.GinValid
type GinValid struct {
    Code  int
    Check int
    Type  int  //GetPost/1 Array/2 Body/3 Header/4 Cookie/5 File/6 Param/7 Query/8 PostForm/9
    Msg   string
}
//全局初始化gin.Context  使用方法  Wsy.GinContext
type GinContext = gin.Context
// Init 初始化并启动 Web 服务器
func (g *WsyGin) New() {
	g.Path = Str.IIF(g.Path == "", "website", Fso.IsPath(g.Path))
	g.Port = Str.IIF(g.Port == "", "2512", g.Port)

	gin.SetMode(gin.ReleaseMode) 
    g.engine = gin.New()
    g.SetMiddleware()
    g.SetRoutes()
    // 启动服务器
    Logs("INFO", "WSYGIN", "[%v] WEB服务启动成功,端口:%v,目录:%v", Date.Now(), g.Port, g.Path, "Y")
    g.engine.Run(":" + g.Port)
}

// setupMiddleware 设置中间件
func (g *WsyGin) SetMiddleware() {
	var SetInfo io.Writer
	if g.Debug {
		SetInfo = gin.DefaultWriter
	} else {
		SetInfo = io.Discard
	}
    g.engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
        Formatter: func(param gin.LogFormatterParams) string {
            return "[WEBSITE]" +
                " " + param.TimeStamp.Format("2006-01-02 15:04:05") + 
                " | IP:" + param.ClientIP +
                " | 用时:" + param.Latency.String() +
                " | 状态:" + Str.ToString(param.StatusCode) +
                " | 响应体:" + Str.ToString(param.BodySize) + "字节" + 
                " | 方法:" + param.Method +
                " | 路径:" + param.Path + "\n"
        },
        Output: SetInfo,
    }), gin.Recovery())
        g.engine.Use(g.SetHeader())
        g.engine.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
        g.SetHtml(c, 500)
    }))
}

// setupRoutes 设置路由
func (g *WsyGin) SetRoutes() { 
    // 静态文件服务
	if g.Dirs {
		g.engine.StaticFS("/", http.Dir(g.Path))
	} else {
		g.engine.Static("/", g.Path)
	}    
    // 404 处理
    g.engine.NoRoute(func(c *gin.Context) {
        g.SetHtml(c, 404)
    })
    // 405 处理
    g.engine.NoMethod(func(c *gin.Context) {
        g.SetHtml(c, 405)
    })
}

// AccessControl 跨域控制中间件
func (g *WsyGin) SetHeader() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Auth-Token, Authorization, X-Requested-With, Accept, Origin, User-Agent")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Max-Age", "86400")
    }
}

// SetHtml 设置简化的 HTML 错误页面并直接返回响应
func (g *WsyGin) SetHtml(c *gin.Context, statusCode int) {
	html := `
<html>
<head><title>` + Str.ToString(statusCode) + ` Error</title></head>
	<body>
		<center><h1>` + Str.ToString(statusCode) + ` Error</h1></center>
		<hr><center>WSYGO</center>
	</body>
</html>
	`
    c.Header("Content-Type", "text/html; charset=utf-8")
    c.String(statusCode, html)
}
//获取
// 通用参数获取函数，支持 GET（query）和 POST（form）
// errMsg 参数：校验失败时自定义错误信息
// enableXSS：是否开启防XSS处理，默认true
func (g *WsyGin) GetParam(ctx *gin.Context, key string, typeCode int, check int, GetType int, errMsg string, enableXSS ...int) (string, error) {
    var value string
    var valueArray []string
    switch GetType {
        case 1: // 自适合post和get
            QueryForm := ctx.Query(key)
            PostsForm := ctx.PostForm(key)
            value = Str.IIF(QueryForm != "", QueryForm, PostsForm)
        case 2:
            // 自适应数组 post和get
            QueryFormArray := ctx.QueryArray(key)
            PostsFormArray := ctx.PostFormArray(key)
            if len(QueryFormArray) >= 2 || len(PostsFormArray) >= 2 {
                valueArray = Str.IIFS(len(QueryFormArray) >= 2, QueryFormArray, PostsFormArray).([]string)
                return g.GetValidArray(key,valueArray,typeCode,check,errMsg,enableXSS...)
            }else{
                //防止单个
                QueryForm := ctx.Query(key)
                PostsForm := ctx.PostForm(key)
                value = Str.IIF(QueryForm != "", QueryForm, PostsForm)
            }
        case 3: // 请求体
            body, err := io.ReadAll(ctx.Request.Body)
            if err != nil {
                return "", errors.New("读取请求体失败")
            }
            value = string(body)
            enableXSS = []int{1}
        case 4: // 头部
            value = ctx.GetHeader(key)
        case 5: // Cookie
            value,_ = ctx.Cookie(key)
        case 6: // 单个文件上传
            // 获取单个文件
            file, fileErr := ctx.FormFile(key)
            if fileErr != nil {
                return "", errors.New("文件获取失败: " + fileErr.Error())
            }
            if file.Size == 0 {
                return "", errors.New("上传的文件为空")
            }
            fileInfo := map[string]interface{}{
                "filename": file.Filename,
                "size": file.Size,
                "header": file.Header,
                "mimetype": file.Header.Get("Content-Type"),
                "saved": false,
            }
            // 返回文件信息的JSON字符串
            return Json.ToJson(fileInfo), nil
            
    }
    return g.GetValid(key,value,typeCode,check,errMsg,enableXSS...)
}
// 验证复选框
func (g *WsyGin) GetValidArray(key string, value []string, typeCode int, check int, errMsg string, enableXSS ...int) (string,error) {
    if len(value) == 0 {
        return "", errors.New(Str.IIF(errMsg != "", errMsg, key) + "不能为空")
    }
    for i, v := range value {
        isVal := Str.Trim(v)
        if isVal == "" && check == 1 {
            return "", errors.New(Str.IIF(errMsg != "", errMsg, key) + "不能为空")
        }
        if isVal != "" && typeCode > 0 && !Str.Valid(isVal, typeCode) {
            return "", errors.New(Str.IIF(errMsg != "", errMsg, key) + "第" + Str.ToString(i+1) + "个元素格式不正确")
        }
    }
    isValues := Str.ToArrString(value)
    return isValues, nil
}
// 验证数据
func (g *WsyGin) GetValid(key string, value string, typeCode int, check int, errMsg string, enableXSS ...int) (string,error) {
    if value == "" && check == 1 {
        return "", errors.New(Str.IIF(errMsg != "", errMsg, key) + "不能为空")
    }
    if value != "" && typeCode > 0 && !Str.Valid(value, typeCode) {
        return "", errors.New(Str.IIF(errMsg != "", errMsg, key) + "格式不正确")
    }
    if len(enableXSS) == 0 {
        value = Html.Escape(value)
    }
    return value, nil
}
// 获取值
func (g *WsyGin) GetValue(ctx *gin.Context, paramDefs map[string]GinValid) (map[string]string, []string) {
    if _, ok := paramDefs["page"]; !ok {
        paramDefs["page"] = GinValid{3, 0, 1, "page",}
    }
    if _, ok := paramDefs["limit"]; !ok {
        paramDefs["limit"] = GinValid{3, 0, 1, "limit"}
    }
    params := make(map[string]string)
    var errs []string
    for key, def := range paramDefs {
        val, err := g.GetParam(ctx, key, def.Code, def.Check, def.Type, def.Msg)
        if err != nil {
            errs = append(errs, err.Error())
        }
        params[key] = val
    }
    return params, errs
}

// 获取值
// 返回 map[string]interface{} 以支持多种类型（string、map[string]string 等）
func (g *WsyGin) GetPost(ctx *gin.Context, paramDefs map[string]GinValid) (map[string]string, []string) {
    if _, ok := paramDefs["page"]; !ok {
        paramDefs["page"] = GinValid{3, 0, 1, "page",}
    }
    if _, ok := paramDefs["limit"]; !ok {
        paramDefs["limit"] = GinValid{3, 0, 1, "limit"}
    }
    params := make(map[string]string)
    var errs []string
    for key, def := range paramDefs {
        val, err := g.GetParam(ctx, key, def.Code, def.Check, def.Type, def.Msg)
        if err != nil {
            errs = append(errs, err.Error())
        }
        params[key] = val
    }
    return params, errs
}

func (g *WsyGin) GetErr(errs []string) string {
    if len(errs) > 0 {
        return Json.Err(errs[0])
    }
    return ""
}

// GetDBErr 用于统一判断 err 或数据为空的情况
func (g *WsyGin) GetDBErr(err error, data map[string]interface{}, emptyMsg string) string {
    if err != nil {
        return Json.Err(err.Error())
    }
    if len(data) == 0 {
        return Json.Err(emptyMsg)
    }
    return ""
}
// GetHost 获取当前请求的主机名
func (g *WsyGin) GetHost(ctx *gin.Context) string {
    host := ctx.Request.Host
    if host == "" || len(host) == 0 {
        return ""
    }
    return host
}
// GetIP 获取当前请求的IP地址
func (g *WsyGin) GetIP(ctx *gin.Context) string {
    ip := ctx.ClientIP()
    if ip == "" || len(ip) == 0 {
        return ""
    }
    return ip
}
// CheckWebAuth 检查域名授权
func (g *WsyGin) CheckHostAuth(ctx *gin.Context,value string) error {
    //Wsy.Echo(Wsy.Key.Encode("192.168.2.5|api.dev.wsyos.com,2014-12-31"))
    if value == "" || len(value) == 0 {
        return errors.New("授权信息不能为空")
    }
    parts := strings.Split(value, ",")
    host := ctx.Request.Host
    if idx := strings.Index(host, ":"); idx > 0 {
        host = host[:idx]
    }
    domainAuthorized := false
    allowedList := strings.Split(parts[0], "|")
    for _, allowed := range allowedList {
        if allowed == host {
            domainAuthorized = true
            break
        }
    }
    if !domainAuthorized { return errors.New("软件未授权!") }
    if !Date.ToBothTime(Date.Now(),parts[1]) {  return errors.New("软件授权已过期!") }
    return nil
}
