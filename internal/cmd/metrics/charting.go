package metrics

import (
	"github.com/confluentinc/ccloud-sdk-go-v1"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/spf13/cobra"
	"os"
)

func chartResponse(cmd *cobra.Command, response *ccloud.MetricsApiQueryReply) error {
	ch := charts.NewLine()
	ch.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Time",
			Type: "time",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:  "value",
			Scale: true,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{Type: "slider"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: true, Trigger: "axis"}),
	)

	data := make([]opts.LineData, len(response.Result))
	for i, point := range response.Result {
		data[i] = opts.LineData{
			Value: []interface{}{point.Timestamp, point.Value},
		}
	}

	ch.AddSeries("metric", data)

	f, err := os.Create("/tmp/chart.html")
	if err != nil {
		return err
	}
	defer f.Close()

	ch.Render(cmd.OutOrStdout())
	return nil
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
