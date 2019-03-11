package main

import (
	"context"
	"runtime"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"flag"
	"time"

	"github.com/lyft/flinkk8soperator/pkg/apis/app/v1alpha1"
	"github.com/lyft/flinkk8soperator/pkg/controller"
	"github.com/lyft/flinkk8soperator/pkg/controller/common"
	"github.com/lyft/flytestdlib/config"
	"github.com/lyft/flytestdlib/config/viper"
	"github.com/lyft/flytestdlib/logger"
	"github.com/lyft/flytestdlib/promutils"
	"github.com/lyft/flytestdlib/promutils/labeled"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
)

const (
	ResyncPeriodKey  = "resync"
	LogSourceLineKey = "log-source-line"
)

var (
	resyncPeriod  time.Duration
	logSourceLine bool
	cfgFile       string
)

func initConfig(ctx context.Context) error {
	configAccessor := viper.NewAccessor(config.Options{
		StrictMode:  true,
		SearchPaths: []string{cfgFile},
	})

	err := configAccessor.UpdateConfig(ctx)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	flag.DurationVar(&resyncPeriod, ResyncPeriodKey, time.Second*time.Duration(20), "Determines the resync period for all watchers.")
	flag.BoolVar(&logSourceLine, LogSourceLineKey, false, "Logs source code file and line number.")
	flag.StringVar(&cfgFile, "config", "./local_config.yaml", "config file path to load configuration")
}

func printVersion(ctx context.Context) {
	logger.Infof(ctx, "Go Version: %s", runtime.Version())
	logger.Infof(ctx, "Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logger.Infof(ctx, "operator-sdk Version: %v", sdkVersion.Version)
	logger.Infof(ctx, "Resync period: %v", resyncPeriod)
}

func watch(ctx context.Context, resource, kind, namespace string, resyncPeriod time.Duration) {
	watchingNamespace := namespace
	if watchingNamespace == "" {
		watchingNamespace = "*"
	}

	logger.Infof(ctx, "Watching [Resource: %s] [Kind: %s] [Namespace: %s] [SyncPeriod: %v]",
		resource, kind, watchingNamespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
}

func main() {
	flag.Parse()
	ctx := context.Background()
	err := initConfig(ctx)
	if err != nil {
		logger.Errorf(ctx, "Unable to load config %v", err)
		os.Exit(1)
	}
	printVersion(ctx)

	sdk.ExposeMetricsPort()

	if logSourceLine {
		logger.SetConfig(logger.Config{IncludeSourceCode: logSourceLine})
		logger.Warn(ctx, "Logging source lines. This might have performance implications.")
	}

	resource := v1alpha1.SchemeGroupVersion.String()
	kind := v1alpha1.FlinkApplicationKind
	namespace, _ := k8sutil.GetWatchNamespace()
	watch(ctx, resource, kind, namespace, resyncPeriod)
	operatorScope := promutils.NewScope("flinkk8soperator")
	labeled.SetMetricKeys(common.GetValidLabelNames()...)
	sdk.Handle(controller.NewHandler(operatorScope))
	sdk.Run(context.TODO())
}
