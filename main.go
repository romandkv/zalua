package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/tealeg/xlsx"
)

type Row struct {
	TaskID               string `xlsx:"6"`
	TimeInProgressInDays string `xlsx:"9"`
}

func main() {
	// Define flags
	token := flag.String("token", "", "a string")
	filename := flag.String("filename", "", "an int")
	sheetFlag := flag.String("sheet", "", "a bool")
	year := flag.Int("year", -1, "a bool")
	month := flag.Int("month", -1, "a bool")
	flag.Parse()
	if year == nil || month == nil || *year == -1 || *month == -1 {
		fmt.Println("Usage: ./main -token=<token> -filename=<filename> -sheet=<sheet> -year=<year> -month=<month>")
		return
	}
	if *token == "" || *filename == "" || *sheetFlag == "" {
		fmt.Println("Usage: ./main -token=<token> -filename=<filename> -sheet=<sheet> -year=<year> -month=<month>")
		return
	}
	startOfGivenMonth := time.Date(*year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
	endOfGivenMonth := time.Date(*year, time.Month(*month), 0, 0, 0, 0, 0, time.UTC).
		AddDate(0, 1, 0).
		Add(-time.Second)
	tp := jira.BearerAuthTransport{
		Token: *token,
	}
	jiraClient, err := jira.NewClient(tp.Client(), "https://jira.softswiss.net")
	if err != nil {
		fmt.Printf("Error creating JIRA client: %v\n", err)
		return
	}
	file, err := xlsx.OpenFile(*filename)
	if err != nil {
		fmt.Printf("Error opening XLSX file: %v\n", err)
		return
	}
	sheet, ok := file.Sheet[*sheetFlag]
	if !ok {
		fmt.Printf("Sheet %s not found\n", os.Args[2])
		return
	}
	for i := 1; i < len(sheet.Rows); i++ {
		row := Row{}
		err := sheet.Row(i).ReadStruct(&row)
		if err != nil {
			fmt.Printf("Error reading row: %v\n", err)
			return
		}
		if row.TaskID == "" {
			continue
		}
		issue, response, err := jiraClient.Issue.Get(row.TaskID, nil)
		if err != nil {
			fmt.Printf("Error getting issue: %v %s\n ", err, row.TaskID)
			return
		}
		if response.StatusCode != 200 {
			fmt.Printf("Error getting issue: %v %s\n", response.Status)
			return
		}
		if issue.Changelog == nil {
			continue
		}
		timeInProgress := time.Duration(0)
		lastInProgressStartTime := time.Time{}
		for _, history := range issue.Changelog.Histories {
			historyCreatedTime, err := time.Parse("2006-01-02T15:04:05.999-0700", history.Created)
			if err != nil {
				fmt.Printf("Error parsing time: %v\n in task %s", err, row.TaskID)
				return
			}
			for _, item := range history.Items {
				if item.Field != "status" {
					continue
				}
				if item.ToString == "In Progress" {
					lastInProgressStartTime = historyCreatedTime
					break
				}
				if item.FromString == "In Progress" {
					if historyCreatedTime.Before(startOfGivenMonth) {
						break
					}
					if row.TaskID == "JSA-3358" {
						fmt.Println("JSA-3358")
					}
					if historyCreatedTime.After(endOfGivenMonth) {
						timeInProgress += endOfGivenMonth.Sub(lastInProgressStartTime)
						break
					}
					if lastInProgressStartTime.Before(startOfGivenMonth) {
						timeInProgress += historyCreatedTime.UTC().Sub(startOfGivenMonth)
						break
					}
					timeInProgress += historyCreatedTime.Sub(lastInProgressStartTime)
					break
				}
			}
		}
		expectedTime := float64(timeInProgress) / 24 / float64(time.Hour)
		givenTimeInProgress, err := strconv.ParseFloat(row.TimeInProgressInDays, 64)
		if err != nil {
			fmt.Printf("%sKO Task %s%s parsing given time: %v\n", RED, row.TaskID, RESET, err)
			continue
		}
		if math.Abs(givenTimeInProgress-expectedTime) > 0.01 {
			fmt.Printf("%sKO Task %s%s: expected time %s, given time %s\n",
				RED,
				RESET,
				row.TaskID,
				strconv.FormatFloat(float64(timeInProgress)/24/float64(time.Hour), 'f', 6, 32),
				row.TimeInProgressInDays,
			)
		} else {
			fmt.Printf("%sOK Task %s%s: expected time %s, given time %s\n",
				GREEN,
				RESET,
				row.TaskID,
				strconv.FormatFloat(float64(timeInProgress)/24/float64(time.Hour), 'f', 6, 32),
				row.TimeInProgressInDays,
			)
		}
	}
}

const (
	RED   = "\033[0;31m"
	GREEN = "\033[0;32m"
	RESET = "\033[0m"
)
