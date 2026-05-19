package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"upay_pro/cron"
	"upay_pro/mylog"
	"upay_pro/web"

	"go.uber.org/zap"
)

const (
	EnvWorkerKey    = "UPAY_IS_WORKER"
	EnvWorkerVal    = "1"
	ExitCodeRestart = 100
)

func main() {
	// 检查是否运行在 Worker 模式
	if os.Getenv(EnvWorkerKey) == EnvWorkerVal {
		runWorker()
		return
	}

	// 检查是否包含 daemon 启动参数
	var daemonMode bool
	for _, arg := range os.Args[1:] {
		if arg == "-d" || arg == "--daemon" {
			daemonMode = true
			break
		}
	}

	if daemonMode {
		// 如果指定了 -d，但当前不是 daemon 主进程，则 fork 启动守护进程
		if os.Getenv("UPAY_DAEMON") != "1" {
			runDaemon()
			return
		}
	}

	// 运行守护主进程
	runSupervisor()
}

func runDaemon() {
	// 过滤掉 -d/--daemon 参数传递给子进程
	var args []string
	for _, arg := range os.Args[1:] {
		if arg != "-d" && arg != "--daemon" {
			args = append(args, arg)
		}
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "UPAY_DAEMON=1")

	// 将后台守护进程的输出重定向到 upay_daemon.log 文件中以备查看
	logFile, err := os.OpenFile("upay_daemon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	setDaemonSysProcAttr(cmd)

	err = cmd.Start()
	if err != nil {
		fmt.Printf("后台守护进程启动失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("UPay Pro 已在后台启动 (PID: %d)\n", cmd.Process.Pid)
	os.Exit(0)
}

func runWorker() {
	defer func() {
		if err := recover(); err != nil {
			mylog.Logger.Error("工作子进程发生恐慌导致崩溃", zap.Any("error", err))
			os.Exit(1)
		}
	}()

	go cron.Start()
	web.Start()
}

func runSupervisor() {
	mylog.Logger.Info("守护主进程已启动，正在启动并监控工作进程...")

	for {
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", EnvWorkerKey, EnvWorkerVal))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// 监听信号并转发给子进程
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		err := cmd.Start()
		if err != nil {
			mylog.Logger.Error("启动工作子进程失败，将在 5 秒后重试", zap.Error(err))
			time.Sleep(5 * time.Second)
			signal.Stop(sigChan)
			continue
		}

		mylog.Logger.Info("已成功启动工作子进程", zap.Int("pid", cmd.Process.Pid))

		// 监控信号的协程
		go func() {
			sig, ok := <-sigChan
			if !ok {
				return
			}
			if cmd.Process != nil {
				mylog.Logger.Info("守护主进程接收到退出信号，正在终止工作进程...", zap.String("signal", sig.String()))
				_ = cmd.Process.Kill()
			}
			os.Exit(0)
		}()

		err = cmd.Wait()
		signal.Stop(sigChan)
		close(sigChan)

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode := exitErr.ExitCode()
				if exitCode == ExitCodeRestart {
					mylog.Logger.Info("检测到配置修改，工作子进程请求重启...")
					time.Sleep(1 * time.Second) // 稍作停顿以确保资源释放
					continue
				}
				mylog.Logger.Error("工作子进程异常退出", zap.Int("exit_code", exitCode))
			} else {
				mylog.Logger.Error("等待工作子进程结束时发生错误", zap.Error(err))
			}
		} else {
			mylog.Logger.Info("工作子进程正常退出，主进程将退出")
			break
		}

		mylog.Logger.Info("主进程将在 3 秒后重新拉起工作进程...")
		time.Sleep(3 * time.Second)
	}
}
