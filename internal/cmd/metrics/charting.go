package metrics

import (
	"fmt"
	"github.com/confluentinc/ccloud-sdk-go-v1"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/spf13/cobra"
	"strings"
)

func chartResponseAsHtml(cmd *cobra.Command, query *ccloud.MetricsApiRequest, response abstractMetricsApiQueryReply) error {
	chart := charts.NewLine()
	chart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:     "1200px",
			Height:    "900px",
			PageTitle: "Confluent Cloud Metrics",
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Metric",
			Subtitle: query.Aggregations[0].Metric,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   true,
			Type:   "scroll",
			Orient: "vertical",
			Right:  "0",
		}),
		charts.WithGridOpts(opts.Grid{
			Left:   "",
			Right:  "350px",
			Top:    "",
			Bottom: "",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Time",
			Type: "time",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:  "value",
			Scale: true,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{}),
		charts.WithTooltipOpts(opts.Tooltip{Show: true, Trigger: "axis"}),
	)

	switch resp := response.(type) {
	case *ccloud.MetricsApiQueryReply:
		{
			data := make([]opts.LineData, len(resp.Result))
			for i, point := range resp.Result {
				data[i] = opts.LineData{
					Value: []interface{}{point.Timestamp, point.Value},
				}
			}
			chart.AddSeries("metric", data)
		}
	case *ccloud.MetricsApiQueryGroupedReply:
		{
			for _, group := range resp.Result {
				data := make([]opts.LineData, len(group.Points))
				for i, point := range group.Points {
					data[i] = opts.LineData{
						Value: []interface{}{point.Timestamp, point.Value},
					}
				}
				arr := make([]string, 0)
				for k, v := range group.Labels {
					arr = append(arr, fmt.Sprintf("%s=%s", k, v))
				}
				chart.AddSeries(strings.Join(arr, ", "), data)
			}
		}
	}

	page := components.NewPage()
	page.AddCharts(chart)
	page.SetLayout(components.PageCenterLayout)

	return page.Render(cmd.OutOrStdout())
}

// https://github.com/wcharczuk/go-chart
//func chartResponseGoChart(response *ccloud.MetricsApiQueryReply) {
//	xValues := make([]time.Time, len(response.Result))
//	yValues := make([]float64, len(response.Result))
//
//	for i, point := range response.Result {
//		xValues[i] = point.Timestamp
//		yValues[i] = point.Value
//	}
//
//	ch := chart.Chart{
//		XAxis: chart.XAxis{
//			ValueFormatter: chart.TimeMinuteValueFormatter,
//		},
//		YAxis: chart.YAxis{
//			AxisType: chart.YAxisSecondary,
//		},
//		Series: []chart.Series{
//			chart.TimeSeries{
//				XValues: xValues,
//				YValues: yValues,
//			},
//		},
//	}
//
//	f, err := os.Create("/tmp/chart.png")
//	if err != nil {
//		panic(err)
//	}
//	defer f.Close()
//
//	ch.Render(chart.PNG, f)
//}

// https://github.com/gonum/plot
//func chartResponseGoNumPlot(response *ccloud.MetricsApiQueryReply) {
//	p := plot.New()
//	xticks := plot.TimeTicks{}
//
//	p.Title.Text = "Example"
//	p.X.Label.Text = "Time"
//	p.X.Tick.Marker = xticks
//	p.Add(plotter.NewGrid())
//	//p.Y.Label.Text = "Y"
//
//	xys := make(plotter.XYs, len(response.Result))
//
//	for i, point := range response.Result {
//		xys[i] = plotter.XY{
//			X: float64(point.Timestamp.Unix()),
//			Y: point.Value,
//		}
//	}
//	spew.Dump(xys)
//
//	err := plotutil.AddLinePoints(p, "metric", xys)
//	if err != nil {
//		panic(err)
//	}
//
//	// Save the plot to a PNG file.
//	if err := p.Save(4*vg.Inch, 4*vg.Inch, "/tmp/chart.png"); err != nil {
//		panic(err)
//	}
//}
