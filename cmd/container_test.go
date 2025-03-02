package cmd

import (
	"context"
	"github.com/numary/ledger/pkg/ledgertesting"
	"github.com/numary/ledger/pkg/opentelemetry/opentelemetrytraces"
	"github.com/numary/ledger/pkg/storage"
	"github.com/numary/ledger/pkg/storage/sqlstorage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestContainers(t *testing.T) {

	pgServer, err := ledgertesting.PostgresServer()
	assert.NoError(t, err)
	defer pgServer.Close()

	type testCase struct {
		name    string
		options []fx.Option
		init    func(*viper.Viper)
	}

	for _, tc := range []testCase{
		{
			name: "default",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.SQLite.String())
			},
		},
		{
			name: "default-with-opentelemetry-traces-on-stdout",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.SQLite.String())
				v.Set(otelTracesFlag, true)
				v.Set(otelTracesExporterFlag, "stdout")
			},
			options: []fx.Option{
				fx.Invoke(fx.Annotate(func(t *testing.T, exp trace.SpanExporter, options ...trace.TracerProviderOption) {
					assert.Len(t, options, 2)
					if os.Getenv("CI") == "true" { // runtime.FuncForPC does not returns same results locally or in the CI pipeline (probably related to inlining)
						return
					}
					var (
						foundWithResource bool
						foundWithSyncer   bool
					)
					for _, opt := range options {
						if strings.Contains(runtime.FuncForPC(reflect.ValueOf(opt).Pointer()).Name(), "trace.WithSyncer") {
							foundWithSyncer = true
						}
						if strings.Contains(runtime.FuncForPC(reflect.ValueOf(opt).Pointer()).Name(), "trace.WithResource") {
							foundWithResource = true
						}
					}
					assert.True(t, foundWithResource)
					assert.True(t, foundWithSyncer)
				}, fx.ParamTags(``, ``, opentelemetrytraces.TracerProviderOptionKey))),
			},
		},
		{
			name: "default-with-opentelemetry-traces-on-stdout-and-batch",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.SQLite.String())
				v.Set(otelTracesFlag, true)
				v.Set(otelTracesExporterFlag, "stdout")
				v.Set(otelTracesBatchFlag, true)
			},
			options: []fx.Option{
				fx.Invoke(fx.Annotate(func(t *testing.T, exp trace.SpanExporter, options ...trace.TracerProviderOption) {
					assert.Len(t, options, 2)
					if os.Getenv("CI") == "true" { // runtime.FuncForPC does not returns same results locally or in the CI pipeline (probably related to inlining)
						return
					}
					var (
						foundWithResource bool
						foundWithBatcher  bool
					)
					for _, opt := range options {
						if strings.Contains(runtime.FuncForPC(reflect.ValueOf(opt).Pointer()).Name(), "trace.WithBatch") {
							foundWithBatcher = true
						}
						if strings.Contains(runtime.FuncForPC(reflect.ValueOf(opt).Pointer()).Name(), "trace.WithResource") {
							foundWithResource = true
						}
					}
					assert.True(t, foundWithResource)
					assert.True(t, foundWithBatcher)
				}, fx.ParamTags(``, ``, opentelemetrytraces.TracerProviderOptionKey))),
			},
		},
		{
			name: "default-with-opentelemetry-traces-on-otlp",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.SQLite.String())
				v.Set(otelTracesFlag, true)
				v.Set(otelTracesExporterFlag, "otlp")
			},
		},
		{
			name: "default-with-opentelemetry-traces-on-jaeger",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.SQLite.String())
				v.Set(otelTracesFlag, true)
				v.Set(otelTracesExporterFlag, "jaeger")
			},
		},
		{
			name: "default-with-opentelemetry-metrics-on-stdout",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.SQLite.String())
				v.Set(otelMetricsFlag, true)
				v.Set(otelMetricsExporterFlag, "stdout")
			},
		},
		{
			name: "default-with-opentelemetry-metrics-on-otlp",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.SQLite.String())
				v.Set(otelMetricsFlag, true)
				v.Set(otelMetricsExporterFlag, "otlp")
			},
		},
		{
			name: "pg",
			init: func(v *viper.Viper) {
				v.Set(storageDriverFlag, sqlstorage.PostgreSQL.String())
				v.Set(storagePostgresConnectionStringFlag, pgServer.ConnString())
			},
			options: []fx.Option{
				fx.Invoke(func(t *testing.T, storageFactory storage.Factory) {
					store, err := storageFactory.GetStore("testing")
					assert.NoError(t, err)
					assert.NoError(t, store.Close(context.Background()))
				}),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := make(chan struct{}, 1)
			options := append(tc.options,
				fx.Invoke(func() {
					run <- struct{}{}
				}),
				fx.Provide(func() *testing.T {
					return t
				}),
			)
			v := viper.New()
			tc.init(v)
			app := NewContainer(v, options...)

			err := app.Start(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer app.Stop(context.Background())

			select {
			case <-run:
			default:
				t.Fatal("application not started correctly")
			}
		})
	}

}
