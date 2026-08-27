package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wetland-release-workbench/internal/assessment"
	"wetland-release-workbench/internal/store"
	"wetland-release-workbench/internal/web"
	"wetland-release-workbench/internal/workflow"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Printf("服务退出：%v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	database := cfg.database
	if cfg.selfcheck {
		database = "file:selfcheck?mode=memory&cache=shared"
	}
	persistence, err := store.Open(database)
	if err != nil {
		return err
	}
	defer persistence.Close()
	recovery, err := persistence.RecoverySummary(context.Background())
	if err != nil {
		return err
	}
	if recovery.UnfinishedCount > 0 {
		log.Printf("已从 schemaVersion %d 恢复 %d 个未完成批次", recovery.SchemaVersion, recovery.UnfinishedCount)
	}
	signer := assessment.NewSigner("wetland-local-v1", []byte("benzhi-wetland-release-local-signing-key-v1"))
	service := workflow.New(persistence, signer)
	handler := web.New(service)
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.addr, err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 45 * time.Second, MaxHeaderBytes: 1 << 20}
	errCh := make(chan error, 1)
	go func() {
		serveErr := server.Serve(listener)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
		close(errCh)
	}()
	if cfg.selfcheck {
		return runCheck(server, errCh, listener.Addr().String())
	}
	log.Printf("湿地修复苗木投放核验工作台已启动：http://%s", listener.Addr().String())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case sig := <-signals:
		log.Printf("收到 %s，开始优雅关闭", sig)
	case serveErr := <-errCh:
		if serveErr != nil {
			return serveErr
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

func runCheck(server *http.Server, errCh <-chan error, addr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	checkErr := web.RunSelfCheck(ctx, "http://"+addr)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	serveErr := <-errCh
	if checkErr != nil {
		return checkErr
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	if serveErr != nil {
		return serveErr
	}
	log.Printf("selfcheck 通过：完成建档、证据、方案、观察、批准与凭据验证")
	return nil
}
