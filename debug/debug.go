package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/xiusin/router/core"
	"github.com/xiusin/router/core/components/di"
)

var l sync.Once

type errHandler struct {
	core.ErrHandler
	fileContent   []string
	firstFileCode string
	firstLine     int
}

func New(r *core.Router) *errHandler {
	l.Do(func() {
		_, f, _, _ := runtime.Caller(0)
		r.Static("/debug_static", path.Dir(f)+"/assets")
	})
	return &errHandler{}
}
func (e *errHandler) Recover(c *core.Context) func() {
	return func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			if di.Exists(di.LOGGER) {
				c.Logger().Printf(
					"msg: %s  Method: %s  Path: %s\n Stack: %s",
					err,
					c.Request().Method,
					c.Request().URL.Path,
					stack,
				)
			}
			e.errors(c, fmt.Sprintf("%s", err), e.showTraceInfo(string(stack)))
		}
	}
}

func (e *errHandler) errors(c *core.Context, errmsg, trace string) {
	c.SetStatus(500)
	_, f, _, _ := runtime.Caller(0)
	tpl, err := template.ParseFiles(path.Dir(f) + "/assets/debug.html")
	if err != nil {
		panic(err.Error())
		return
	}
	jsData, _ := json.Marshal(e.fileContent)
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]interface{}{
		"stack":     template.HTML(trace),
		"error":     errmsg,
		"fileMap":   string(jsData),
		"firstLine": strconv.Itoa(e.firstLine),
		"firstCode": e.firstFileCode,
	}); err != nil {
		panic(err.Error())
	}
	_, _ = c.Writer().Write(buf.Bytes())
}

func (e *errHandler) showTraceInfo(msg string) string {
	msgs := strings.Split(strings.Trim(msg, "\n"), "\n")
	var str string
	msgs = msgs[1:]
	l := len(msgs)
	idx := 1
	var fileContentMap []string
	for i := 0; i < l; i += 2 {
		paths := strings.Split(msgs[i+1], ":")
		paths[0] = strings.Trim(paths[0], "\t")
		// 读取文件内容
		codeContent, _ := ioutil.ReadFile(paths[0])
		line := strings.Split(paths[1], " ")
		lineNum, _ := strconv.Atoi(line[0])
		codes := strings.Split(string(codeContent), "\n")
		ln, _ := strconv.Atoi(line[0])
		codes[ln-1] = codes[ln-1] + "	  //	 <-----  堆栈调用位置"
		count := len(codes)
		var firstLine int

		if count-lineNum < 25 && count-50 > 0 {
			firstLine = count - 50
			codes = codes[count-50:]
		} else if lineNum < 25 && count > 50 {
			codes = codes[:]
			firstLine = 0
		} else {
			var start int
			var end int
			if lineNum > 25 {
				start = lineNum - 25
			}
			if lineNum+25 > count {
				end = count
			} else {
				end = lineNum + 25
			}
			firstLine = start
			codes = codes[start:end]
		}
		s := strings.Join(codes, "\n")
		fileContentMap = append(fileContentMap, s)

		str += `<div class="__BtrD__loop-tog __BtrD__l-parent" data-id="proc-` +
			strconv.Itoa(idx) + `" title="_GLOBAL" data-file="` +
			paths[0] + `" data-class="trigger_error" data-fline="` + strconv.Itoa(firstLine) + `" data-line="` +
			line[0] + `"><div class="__BtrD__id __BtrD__loop-tog __BtrD__code">` +
			strconv.Itoa(idx) + `</div><div class="__BtrD__holder"><span class="__BtrD__name">` +
			msgs[i] + `</b><i class="__BtrD__line">` +
			line[0] + `</i></span><span class="__BtrD__path">` +
			paths[0] + `</span></div></div>`
		idx++
		if e.firstFileCode == "" {
			e.firstFileCode = s
			e.firstLine = firstLine + 1
		}
	}
	e.fileContent = fileContentMap
	return str
}
