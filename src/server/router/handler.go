package router

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/errorx"
	"github.com/toolkits/pkg/ginx"
	"github.com/ulricqin/ibex/src/models"
	"github.com/ulricqin/ibex/src/server/config"
)

func taskStdout(c *gin.Context) {
	meta := TaskMeta(ginx.UrlParamInt64(c, "id"))
	stdouts, err := meta.Stdouts()
	ginx.NewRender(c).Data(stdouts, err)
}

func taskStderr(c *gin.Context) {
	meta := TaskMeta(ginx.UrlParamInt64(c, "id"))
	stderrs, err := meta.Stderrs()
	ginx.NewRender(c).Data(stderrs, err)
}

// TODO: 不能只判断task_action，还应该看所有的host执行情况
func taskState(c *gin.Context) {
	action, err := models.TaskActionGet("id=?", ginx.UrlParamInt64(c, "id"))
	if err != nil {
		ginx.NewRender(c).Data("", err)
		return
	}

	state := "done"
	if action != nil {
		state = action.Action
	}

	ginx.NewRender(c).Data(state, err)
}

func taskResult(c *gin.Context) {
	id := ginx.UrlParamInt64(c, "id")

	hosts, err := models.TaskHostStatus(id)
	if err != nil {
		errorx.Bomb(500, "load task hosts of %d occur error %v", id, err)
	}

	ss := make(map[string][]string)
	total := len(hosts)
	for i := 0; i < total; i++ {
		s := hosts[i].Status
		ss[s] = append(ss[s], hosts[i].Host)
	}

	ginx.NewRender(c).Data(ss, nil)
}

func taskHostOutput(c *gin.Context) {
	obj, err := models.TaskHostGet(ginx.UrlParamInt64(c, "id"), ginx.UrlParamStr(c, "host"))
	ginx.NewRender(c).Data(obj, err)
}

func taskHostStdout(c *gin.Context) {
	id := ginx.UrlParamInt64(c, "id")
	host := ginx.UrlParamStr(c, "host")

	if config.C.Output.ComeFrom == "database" || config.C.Output.ComeFrom == "" {
		obj, err := models.TaskHostGet(id, host)
		ginx.NewRender(c).Data(obj.Stdout, err)
		return
	}

	if config.C.Output.AgtdPort <= 0 || config.C.Output.AgtdPort > 65535 {
		ginx.NewRender(c).Message(fmt.Errorf("remotePort(%d) invalid", config.C.Output.AgtdPort))
		return
	}

	url := fmt.Sprintf("http://%s:%d/output/%d/stdout.json", host, config.C.Output.AgtdPort, id)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(url)
	errorx.Dangerous(err)

	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	errorx.Dangerous(err)

	c.Writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	c.Writer.Write(bs)
}

func taskHostStderr(c *gin.Context) {
	id := ginx.UrlParamInt64(c, "id")
	host := ginx.UrlParamStr(c, "host")

	if config.C.Output.ComeFrom == "database" || config.C.Output.ComeFrom == "" {
		obj, err := models.TaskHostGet(id, host)
		ginx.NewRender(c).Data(obj.Stderr, err)
		return
	}

	if config.C.Output.AgtdPort <= 0 || config.C.Output.AgtdPort > 65535 {
		ginx.NewRender(c).Message(fmt.Errorf("remotePort(%d) invalid", config.C.Output.AgtdPort))
		return
	}

	url := fmt.Sprintf("http://%s:%d/output/%d/stderr.json", host, config.C.Output.AgtdPort, id)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(url)
	errorx.Dangerous(err)

	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	errorx.Dangerous(err)

	c.Writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	c.Writer.Write(bs)
}

func taskStdoutTxt(c *gin.Context) {
	id := ginx.UrlParamInt64(c, "id")

	meta, err := models.TaskMetaGet("id = ?", id)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	if meta == nil {
		c.String(404, "no such task")
		return
	}

	stdouts, err := meta.Stdouts()
	if err != nil {
		c.String(500, err.Error())
		return
	}

	w := c.Writer

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	count := len(stdouts)
	for i := 0; i < count; i++ {
		if i != 0 {
			w.Write([]byte("\n\n"))
		}

		w.Write([]byte(stdouts[i].Host + ":\n"))
		w.Write([]byte(stdouts[i].Stdout))
	}
}

func taskStderrTxt(c *gin.Context) {
	id := ginx.UrlParamInt64(c, "id")

	meta, err := models.TaskMetaGet("id = ?", id)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	if meta == nil {
		c.String(404, "no such task")
		return
	}

	stderrs, err := meta.Stderrs()
	if err != nil {
		c.String(500, err.Error())
		return
	}

	w := c.Writer

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	count := len(stderrs)
	for i := 0; i < count; i++ {
		if i != 0 {
			w.Write([]byte("\n\n"))
		}

		w.Write([]byte(stderrs[i].Host + ":\n"))
		w.Write([]byte(stderrs[i].Stderr))
	}
}

type TaskStdoutData struct {
	Host   string `json:"host"`
	Stdout string `json:"stdout"`
}

type TaskStderrData struct {
	Host   string `json:"host"`
	Stderr string `json:"stderr"`
}

func taskStdoutJSON(c *gin.Context) {
	task := TaskMeta(ginx.UrlParamInt64(c, "id"))

	host := ginx.QueryStr(c, "host", "")

	var hostsLen int
	var ret []TaskStdoutData

	if host != "" {
		obj, err := models.TaskHostGet(task.Id, host)
		if err != nil {
			ginx.NewRender(c).Data("", err)
			return
		} else if obj == nil {
			ginx.NewRender(c).Data("", fmt.Errorf("task: %d, host(%s) not eixsts", task.Id, host))
			return
		} else {
			ret = append(ret, TaskStdoutData{
				Host:   host,
				Stdout: obj.Stdout,
			})
		}
	} else {
		hosts, err := models.TaskHostGets(task.Id)
		if err != nil {
			ginx.NewRender(c).Data("", err)
			return
		}

		hostsLen = len(hosts)

		ret = make([]TaskStdoutData, 0, hostsLen)
		for i := 0; i < hostsLen; i++ {
			ret = append(ret, TaskStdoutData{
				Host:   hosts[i].Host,
				Stdout: hosts[i].Stdout,
			})
		}
	}

	ginx.NewRender(c).Data(ret, nil)
}

func taskStderrJSON(c *gin.Context) {
	task := TaskMeta(ginx.UrlParamInt64(c, "id"))

	host := ginx.QueryStr(c, "host", "")

	var hostsLen int
	var ret []TaskStderrData

	if host != "" {
		obj, err := models.TaskHostGet(task.Id, host)
		if err != nil {
			ginx.NewRender(c).Data("", err)
			return
		} else if obj == nil {
			ginx.NewRender(c).Data("", fmt.Errorf("task: %d, host(%s) not eixsts", task.Id, host))
			return
		} else {
			ret = append(ret, TaskStderrData{
				Host:   host,
				Stderr: obj.Stderr,
			})
		}
	} else {
		hosts, err := models.TaskHostGets(task.Id)
		if err != nil {
			ginx.NewRender(c).Data("", err)
			return
		}

		hostsLen = len(hosts)

		ret = make([]TaskStderrData, 0, hostsLen)
		for i := 0; i < hostsLen; i++ {
			ret = append(ret, TaskStderrData{
				Host:   hosts[i].Host,
				Stderr: hosts[i].Stderr,
			})
		}
	}

	ginx.NewRender(c).Data(ret, nil)
}
