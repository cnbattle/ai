package ai

import (
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

var LOG = logrus.New()

func init() {
	// 设置输出为标准输出
	LOG.Out = os.Stdout
	// 设置日志级别为 Debug
	LOG.SetLevel(logrus.TraceLevel)
	// logrus.SetFormatter(&logrus.JSONFormatter{})

	// 设置输出格式为 TextFormatter
	// logger.SetFormatter(&logrus.TextFormatter{
	// 	// FullTimestamp: true,
	// })

	LOG.SetReportCaller(true)
	LOG.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		// ForceColors:     true, // 强制使用颜色
		TimestampFormat: "2006-01-02 15:04:05.000",
		CallerPrettyfier: func(f *runtime.Frame) (second string, first string) {
			return "", fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			// _, b, _, _ := runtime.Caller(0)
			// basepath := filepath.Dir(b)
			// rel, err := filepath.Rel(basepath, f.File)
			// if err != nil {
			// 	LOG.Error("Couldn't determine file path\n", err)
			// }
			// return "", fmt.Sprintf("%s:%d", rel, f.Line)
		},
	})
}
