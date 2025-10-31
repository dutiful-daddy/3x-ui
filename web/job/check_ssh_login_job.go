package job

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v2/web/service"
)

var (
	SSHLoginUser int
	mu           sync.Mutex
)

type CheckSSHLoginJob struct {
	xrayService  service.XrayService
	tgbotService service.Tgbot
}

func NewCheckSSHLoginJob() *CheckSSHLoginJob {
	return &CheckSSHLoginJob{}
}

func (j *CheckSSHLoginJob) CheckSSHLoginSuccess(startTime time.Time) {
	whoOutput, err := exec.Command("who").Output()
	if err != nil {
		fmt.Println("exec who error:", err)
		return
	}
	lines := strings.Split(strings.TrimSpace(string(whoOutput)), "\n")
	numberInt := len(lines)

	mu.Lock()
	defer mu.Unlock()

	if numberInt > SSHLoginUser {
		SSHLoginUser = numberInt

		if len(lines) == 0 {
			return
		}

		lastLine := lines[len(lines)-1]
		fields := strings.Fields(lastLine)
		if len(fields) < 4 {
			return
		}

		userName := fields[0]
		loginTimeStr := fields[2] + " " + fields[3]
		loginTime, err := time.Parse("2006-01-02 15:04", loginTimeStr)
		if err != nil || loginTime.Before(startTime) {
			fmt.Printf("SSHLogin[%s] early than XRAY-UI start[%s]\n", loginTimeStr, startTime)
			return
		}

		ipAddr := "unknown"
		re := regexp.MustCompile(`\(([^)]+)\)`)
		matches := re.FindStringSubmatch(lastLine)
		if len(matches) > 1 {
			ipAddr = matches[1]
		}

		name, _ := os.Hostname()
		if name == "" {
			name = "Unknown"
		}

		SSHLoginInfo := fmt.Sprintf("新用户登录提醒:\n")
		SSHLoginInfo += fmt.Sprintf("主机名称:%s\n", name)
		SSHLoginInfo += fmt.Sprintf("SSH登录用户:%s", userName)
		SSHLoginInfo += fmt.Sprintf("SSH登录时间:%s", loginTimeStr)
		SSHLoginInfo += fmt.Sprintf("SSH登录IP:%s", ipAddr)
		SSHLoginInfo += fmt.Sprintf("当前SSH登录用户数:%d", numberInt)
		j.tgbotService.SendMsgToTgbotAdmins(SSHLoginInfo)
	} else {
		SSHLoginUser = numberInt
	}
}

func (j *CheckSSHLoginJob) CheckSSHLoginFailed() {
	logFiles := []string{"/var/log/auth.log", "/var/log/secure"}
	var logFile string

	for _, file := range logFiles {
		if _, err := os.Stat(file); err == nil {
			logFile = file
			break
		}
	}

	if logFile == "" {
		return
	}

	fiveMinutesAgo := time.Now().Add(-1 * time.Minute).Format("2006-01-02 15:04:05")

	cmd := fmt.Sprintf("journalctl -u sshd --since '%s' 2>/dev/null | grep -E '(Failed password|Invalid user)'", fiveMinutesAgo)
	output, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		cmd = fmt.Sprintf("grep -E '(Failed password|Invalid user)' %s | grep '%s'", logFile, fiveMinutesAgo[:10])
		output, err = exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		failType, user, ip := parseSSHFailLine(line)
		if user == "" || ip == "" {
			continue
		}

		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "Unknown"
		}

		logTime := extractLogTime(line)

		failMsg := fmt.Sprintf("SSH登录失败提醒:\n"+
			"主机名称: %s\n"+
			"失败类型: %s\n"+
			"尝试登录用户: %s\n"+
			"来源IP: %s\n"+
			"发生时间: %s\n"+
			"请注意服务器安全!", hostname, failType, user, ip, logTime)

		j.tgbotService.SendMsgToTgbotAdmins(failMsg)
	}
}

func extractLogTime(line string) string {
	fields := strings.Fields(line)
	if len(fields) >= 3 {
		return fmt.Sprintf("%s %s %s", fields[0], fields[1], fields[2])
	}
	return time.Now().Format("2006-01-02 15:04:05")
}

func parseSSHFailLine(line string) (failType, user, ip string) {
	if strings.Contains(line, "Failed password for") {
		failType = "密码错误"
		re := regexp.MustCompile(`Failed password for (\S+) from (\S+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 3 {
			user = matches[1]
			ip = matches[2]
		}
	} else if strings.Contains(line, "Invalid user") {
		failType = "无效用户名"
		re := regexp.MustCompile(`Invalid user (\S+) from (\S+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 3 {
			user = matches[1]
			ip = matches[2]
		}
	}
	return
}
