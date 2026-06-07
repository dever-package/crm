package service

type DashboardService struct {
	Metric MetricService
}

func NewDashboardService() DashboardService {
	return DashboardService{Metric: NewMetricService()}
}
