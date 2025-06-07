package handlers


import (
	"context"
	"log"
	"os"
	"strings"
	"time"
	"chatops/internal/monitoring/handlers"
	"github.com/joho/godotenv"
	telebot "gopkg.in/telebot.v3"
)




//metric
func metricHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	service := parts[1]
	metric := parts[2]
	req := metric + "{job=\"" + service + "\"}"
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	response, err := client.Query(ctx, req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("Произошла ошибка: %v", err))
	}
	
	result := fmt.Sprintf("Status: %s\n", response.Status)

	// TODO: че блять 
	for i, res := range response.Data.Result {
    	result += fmt.Sprintf("data.result[%d].value: %v\n", i, res.Value)
	}

	return c.Send(result)

}




//metric
func listMetricsHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	client = NewClient("", "")
	service := parts[1]
	req := service
	metric := parts[2]
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	response, err := client.ListMetrics(ctx, req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("Произошла ошибка: %v", err))
	}
	var matchedMetrics []string
	for _, str := range response {
		if strings.Contains(str, metric) {
			matchedMetrics = append(matchedMetrics, str)
		}
	}

	return c.Send(strings.Join(matchedMetrics, "\n"))
}


