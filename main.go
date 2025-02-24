package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/krunalnaikgo/stockmonitor/constants"
	"github.com/krunalnaikgo/stockmonitor/stocksearch"
	"github.com/krunalnaikgo/stockmonitor/utils"

	"github.com/aws/aws-lambda-go/lambda"
)

type StockEvent struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context) (string, error) {
	return fmt.Sprintf("Stock Event Sent"), nil
}

func main() {
	stockName := utils.GetEnvValue("STOCK")
	apiKey := utils.GetEnvValue("APIKEY")
	dynamodbTableName := utils.GetEnvValue("DYNAMODBTABLE")
	historyTableName := utils.GetEnvValue("HISTORYDBTABLE")
	boughtPrice := utils.GetEnvValue("BOUGHTSTOCKPRICE")
	boughtStockSize := utils.GetEnvValue("BOUGHTSTOCKSIZE")
	stock := stocksearch.StockPriceDetails{StockName: stockName,
		APIKey: apiKey}

	foundOpenVal, foundCloseVal, maxHighPrice := stock.GetStockValues()
	fmt.Println("Got Values", foundOpenVal, foundCloseVal)

	//foundValues := fmt.Sprintf("Stock: %s Open is: "+
	//	"%s and Close is: %s", stockName,
	//	foundOpenVal, foundCloseVal)

	utils.CreateDynamodbTable(constants.AWSREGION, dynamodbTableName, "stockName")
	queryOpen, queryClose, highPrice, _, highPriceDate := utils.QueryTable(constants.AWSREGION,
		dynamodbTableName, stockName)

	checkOpenIncrease := utils.CheckIncreaseValues(queryOpen, foundOpenVal)
	checkCloseIncrease := utils.CheckIncreaseValues(queryClose, foundCloseVal)

	checkHighPrice := utils.CheckIncreaseValues(highPrice, maxHighPrice)

	var maxHighPricedb float64
	var maxHighPriceDatedb string

	if checkHighPrice {
		maxHighPricedb = maxHighPrice
		maxHighPriceDatedb = time.Now().Format("2006-01-02")
	} else {
		maxHighPricedb = highPrice
		maxHighPriceDatedb = highPriceDate
	}
	var textOpenOut string
	var textCloseOut string
	if checkOpenIncrease {
		textOpenOut = fmt.Sprintf("StockName : %s Open Value Increased from : %f to : %f ", stockName,
			queryOpen, foundOpenVal)
	} else {
		textOpenOut = fmt.Sprintf("StockName : %s  Open Value Decreased from : %f to : %f ", stockName,
			queryOpen, foundOpenVal)
	}

	if checkCloseIncrease {
		textCloseOut = fmt.Sprintf("StockName : %s Close Value Increased from : %f to : %f \n", stockName,
			queryClose, foundCloseVal)
	} else {
		textCloseOut = fmt.Sprintf("StockName : %s Close Value Decreased from : %f to : %f \n", stockName,
			queryClose, foundCloseVal)
	}

	log.Println(textOpenOut + "/n")
	log.Println(textCloseOut + "/n")

	strOpenVal := strconv.FormatFloat(foundOpenVal, 'f', 6, 64)
	strCloseVal := strconv.FormatFloat(foundCloseVal, 'f', 6, 64)
	strHighPriceVal := strconv.FormatFloat(maxHighPricedb, 'f', 6, 64)

	floatBoughtPrice, _ := strconv.ParseFloat(strings.TrimSpace(boughtPrice), 64)
	floatBoughtSize, _ := strconv.ParseFloat(strings.TrimSpace(boughtStockSize), 64)
	calcGain := utils.CalculateProfitOrLoss(floatBoughtPrice, maxHighPrice, floatBoughtSize)

	// take Gain value it can be loss or profit for today
	textGainOut := fmt.Sprintf("StockName : %s ,Gain Value For Today is : %f \n", stockName,
		calcGain)

	todayDate := time.Now().Format(constants.TIMEFORMAT)

	// calculate maximum price since buy
	textMaxHighPrice := fmt.Sprintf("StockName : %s , Maxium High Price since Bought is : %f  and Date is :%s \n", stockName,
		maxHighPricedb, maxHighPriceDatedb)

	// It will update table
	utils.UpdateTable(constants.AWSREGION, dynamodbTableName,
		strOpenVal, strCloseVal, strHighPriceVal, stockName,
		maxHighPriceDatedb)

	// create History Table
	utils.CreateDynamodbTable(constants.AWSREGION, historyTableName, "HighPriceDate")
	utils.UpdateHistoryTable(constants.AWSREGION, historyTableName, stockName, todayDate, fmt.Sprintf("%f", calcGain))
	outMap := utils.QueryHistoryTable(constants.AWSREGION, historyTableName, stockName)
	//fmt.Println("OutMap is: Main Routine ", outMap)
	outStringHistory := utils.Get5HistoryDb(outMap)

	finalTextBody := fmt.Sprintf("%s /n %s \n %s \n  %s \n %s \n ", textOpenOut, textCloseOut,
		textGainOut, textMaxHighPrice, outStringHistory)
	toEmail := utils.GetEnvValue("TOEMAIL")
	fromEmail := utils.GetEnvValue("FROMEMAIL")
	notificationSubject := fmt.Sprintf("Stock Notification for Today: %s %s \n\n", stockName, todayDate)
	snsDetailsObj := utils.SNSDetails{
		AwsRegion: "us-east-1",
		FromEmail: fromEmail,
		ToEmail:   toEmail,
		Subject:   notificationSubject,
		CharSet:   "UTF-8",
		TextBody:  finalTextBody,
	}

	//GMAIL
	//sendEmail(toEmail, fromEmail, emailPassword, foundValues)

	//SNS
	snsDetailsObj.SendSNSEmail()
	lambda.Start(HandleRequest)
}
