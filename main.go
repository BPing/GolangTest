package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"github.com/BPing/Golib/httplib"
	"regexp"
	"runtime"
	"strings"
	"syscall"
)

const (
	FFMPEG = "/opt/local/ffmpeg/ffmpeg" ///opt/local/ffmpeg/ffmpeg
	PARARMERR = -1
	LOGFILE = "ffmpeg-log.log"
)

//推流命令控制器
type pushCmd struct {
	CmdExec *exec.Cmd
	Flag    bool //人为正常关闭是否
	Out     *bytes.Buffer
}

//
func (this *pushCmd) ErrorString() (str string) {
	return this.Out.String()
}

//
func (this *pushCmd) Kill() {
	if runtime.GOOS == "linux" {
		pgid, err := syscall.Getpgid(this.CmdExec.Process.Pid)
		if nil == err {
			syscall.Kill(-pgid, syscall.SIGTERM)
		}
		logPrintln(err)
	}
}

var (
	loop bool = false //循环标志
	bitrate string = "1m"  //输出码率设置
	bitReg = regexp.MustCompile(`^(\d){1,6}(k|K|M|m)$`)
	pushReg = regexp.MustCompile(`^rtmp://`)
	avOptions = regexp.MustCompile(`Stream .*\n`)

	pushMap = make(map[string]*pushCmd)

	logger *log.Logger

	callbackServer = "" //接收回调信息的uri

	key = "4fbfc9b4d58838f1757b68c8eff5b5564ba2c7fb"
	authUser = "4G"

	cmdStrf = FFMPEG + " -re -i %s  -c:v copy  %s -f flv  %s"

	cmdInfoStrf = FFMPEG + "  -i %s "
)

func init() {
	fmt.Println("begin Log ...")
	file, err := os.OpenFile(LOGFILE, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0777)
	if err != nil {
		fmt.Println("fail to create " + LOGFILE + " file!")
	}
	logger = log.New(file, "", log.LstdFlags | log.Llongfile)
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////

func logPrintln(v ...interface{}) {
	if nil != logger {
		logger.Println(v...)
	}
	fmt.Println(v...)
}

func jsonEcho(code int, msg interface{}, w http.ResponseWriter) {
	res := make(map[string]interface{})
	res["Code"] = code
	res["Msg"] = msg
	logPrintln(msg)
	bytes, _ := json.Marshal(res)
	w.Write(bytes)

}

///////////////////////////////////////////////////////////////////////////////////////////////////////////

func pushStream(video, pushurl, courseid string) (err error) {
	cmd, in, _ := newCmd()
	cndStr := fmt.Sprintf(cmdStrf, video, "", pushurl)
	in.WriteString(cndStr)
	logPrintln(cndStr)
	err = cmd.Run()
	return
}

func newCmd() (cmd *exec.Cmd, in *bytes.Buffer, out *bytes.Buffer) {
	in = bytes.NewBuffer(nil)
	out = bytes.NewBuffer(nil)
	cmd = exec.Command("sh")
	cmd.Stdin = in
	cmd.Stderr = out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return
}

func newPushCmd(video, pushurl, courseid string) (pCmd *pushCmd) {
	cmd, in, out := newCmd()
	cndStr := fmt.Sprintf(cmdStrf, video, "", pushurl)
	in.WriteString(cndStr)
	logPrintln(cndStr)
	pCmd = &pushCmd{cmd, false, out}
	pushMap[courseid] = pCmd
	return
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////
//头部信息添加签名信息
func Sign(params map[string]string) string {
	return authUser + ":" + sign(params, key)
}

//检查签名
func CheckSign(params map[string]string, signStr string) bool {
	return Sign(params) == signStr
}

func sign(params map[string]string, key string) string {
	str := url.Values{}
	for k, v := range params {
		str.Add(k, v)
	}
	authPwd := fmt.Sprintf("%x", md5.Sum([]byte(str.Encode() + key)))
	return authPwd
}

func HttpGet(url string, params map[string]string) {
	httpreq := httplib.Get(url)
	if nil == httpreq {
		return
	}
	//set params
	for key, val := range params {
		httpreq.Param(key, val)
	}
	httpreq.Header("Accept", "application/json")
	httpreq.Header("Authentication", Sign(params))
	_, err := httpreq.Response()
	if nil != err {
		logPrintln(err)
	}
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////

func LoopHandler(w http.ResponseWriter, r *http.Request) {

	var op string = "close"
	var video string = "http://7xrnmp.com1.z0.glb.clouddn.com/testVideo.mp4"
	var pushurl string = "rtmp://pili-publish.dreamtest.strongwind.cn/DreamtestLive2016/livetest?key=0d281f93-c8ec-4e0f-a573-15680052443f"

	r.ParseForm()
	if len(r.Form["op"]) > 0 {
		op = r.Form["op"][0]
	}

	if len(r.Form["video"]) > 0 {
		video = r.Form["video"][0]
	}

	if len(r.Form["pushurl"]) > 0 {
		pushurl = r.Form["pushurl"][0]
	}

	if !loop && op == "open" {
		loop = true
		go func() {
			for loop {
				logPrintln("push")
				err := pushStream(video, pushurl, "")
				if err != nil {
					fmt.Println(err)
					loop = false
					return
				}
			}
		}()

		jsonEcho(0, "open loop", w)
		return

	} else if op == "close" {
		loop = false
		jsonEcho(0, "close loop", w)
		return
	}
	jsonEcho(0, fmt.Sprintf("nothing to do! the status of loop :\b", loop), w)
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////////////

// @Title PushStream
// @Description 推流请求处理
// @Param   videourl         form   string    true      "视频源地址"
// @Param   pushurl        form   string    true      "推流地址"
// @Param   callback        form   string    true      "回调函数"
// @Param   callback1        form   string    true      "回调函数"
// @Param   courseid        form   string    true      "唯一标识"
func PushStream(w http.ResponseWriter, r *http.Request) {
	logPrintln("PushStream--------------------")
	r.ParseForm()
	if len(r.Form["videourl"]) <= 0 ||
	len(r.Form["pushurl"]) <= 0 ||
	len(r.Form["courseid"]) <= 0 ||
	len(r.Form["callback"]) <= 0 ||
	pushReg.FindString(r.Form["pushurl"][0]) == "" {
		jsonEcho(PARARMERR, "param error", w)
		return
	}
	video := r.Form["videourl"][0]
	pushurl := r.Form["pushurl"][0]
	courseid := r.Form["courseid"][0]
	callback := r.Form["callback"][0]
	callback1 := r.Form["callback1"][0]

	logPrintln(video, pushurl, courseid, callback)

	_, ok := pushMap[courseid]
	if ok {
		jsonEcho(-2, "another stream already exist", w)
		return
	}

	go func() {
		code := "0"
		logPrintln("push:" + courseid)
		cmd := newPushCmd(video, pushurl, courseid)
		err := cmd.CmdExec.Run()
		if err != nil {
			logPrintln(err.Error() + " " + cmd.ErrorString())
			if nil != cmd && !cmd.Flag {
				code = "-1"
			}
		}

		logPrintln("push end:" + courseid)

		delete(pushMap, courseid)

		param := make(map[string]string)
		param["courseid"] = courseid
		param["code"] = code

		if "" != callback1 {
			HttpGet(callback1, param)
		} else {
			HttpGet(callbackServer, param)
		}

		if "" != callback {
			HttpGet(callback, param)
		}
	}()
	jsonEcho(0, "already push ,please waiting for callback", w)
	return
}

//
func Callback(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["code"]) > 0 {
		logPrintln("push code:", string(r.Form["code"][0]))
	}
	logPrintln("push")
	for _, val := range r.Form {
		logPrintln(val[0])
		logPrintln("push")
	}

}

// @Title SetBitRate
// @Description 设置输出码率
// @Param   rate         form   string    true      "唯一标识"
func SetBitRate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["rate"]) > 0 {
		rate := bitReg.FindString(r.Form["rate"][0])
		if "" != rate {
			bitrate = rate
			jsonEcho(0, "success to setting the value of bitrate ", w)
			return
		}

	}
	jsonEcho(PARARMERR, "fail to setting the value of bitrate  ", w)
	return
}

// @Title SetBitRate
// @Description 设置回调地址
// @Param   courseid         form   string    true      "回调地址"
func KillPush(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["courseid"]) <= 0 {
		jsonEcho(PARARMERR, "param error", w)
		return
	}
	courseid := strings.TrimSpace(r.Form["courseid"][0])
	cmd, ok := pushMap[courseid]
	if ok {
		cmd.Flag = true
		cmd.Kill()
	}
	jsonEcho(0, "kill push success:" + courseid, w)
	return
}

// @Title SetBitRate
// @Description 设置回调地址
// @Param   callbackServer         form   string    true      "回调地址"
func SetCallbackServer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["callbackServer"]) <= 0 {
		jsonEcho(PARARMERR, "param error", w)
		return
	}
	callbackServer = r.Form["callbackServer"][0]
	jsonEcho(0, "Set callbackServer success", w)
	return
}


// @Title GetVideoInfo
// @Description 获取视频的基本信息
// @Param   videourl         form   string    true      "视频信息"
func GetVideoInfo(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["videourl"]) <= 0 || r.Form["videourl"][0] == "" {
		jsonEcho(PARARMERR, "param error", w)
		return
	}
	videourl := r.Form["videourl"][0]
	vOpt := make(map[string]string)
	cmd, in, out := newCmd()
	cndStr := fmt.Sprintf(cmdInfoStrf, videourl)
	in.WriteString(cndStr)
	err := cmd.Run()
	if err != nil {
		fmt.Println("GetVideoInfo:" + err.Error())
	}
	str := out.String()
	options := avOptions.FindAllString(str, -1)
	if len(options) == 2 {
		vo := strings.Split(options[0], ",")
		ao := strings.Split(options[1], ",")
		vOpt["V_code"] = vo[0]
		vOpt["V_res"] = vo[2]
		vOpt["V_bitrate"] = vo[3]
		vOpt["V_fps"] = vo[4]

		vOpt["A_code"] = ao[0]
		vOpt["A_samplerate"] = ao[1]
		vOpt["A_bitrate"] = ao[4]

	}
	jsonEcho(0, vOpt, w)
	return
}

//////////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	http.HandleFunc("/loop", LoopHandler)
	http.HandleFunc("/push", PushStream)
	http.HandleFunc("/kill", KillPush)
	http.HandleFunc("/callback", Callback)
	http.HandleFunc("/setrate", SetBitRate)
	http.HandleFunc("/setCallbackServer", SetCallbackServer)
	http.HandleFunc("/videoInfo", GetVideoInfo)
	logPrintln("ListenAndServe:8888")
	http.ListenAndServe(":8888", nil)
}
