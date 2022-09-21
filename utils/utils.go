package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

//StrToDuration is similar to time.ParseDuration() but also supports days "d" and weeks "w"
func StrToDuration(timeStep string) (time.Duration, error) {

	//Defining duration values for each unit
	var unitMap = map[string]int64{
		"ns": int64(time.Nanosecond),
		"us": int64(time.Microsecond),
		"ms": int64(time.Millisecond),
		"s":  int64(time.Second),
		"m":  int64(time.Minute),
		"h":  int64(time.Hour),
		"d":  int64(time.Hour) * 24,
		"w":  int64(time.Hour) * 168,
	}

	if len(timeStep) == 0 {
		return 0, fmt.Errorf("time: invalid duration \"%s\"", timeStep)
	}

	//Starts a loop pointing to the 1st character
	var res int64 = 0
	index := 0
	for index < len(timeStep) {

		//Starts a 2nd loop while digits are detected
		i := index
		for timeStep[i] >= '0' && timeStep[i] <= '9' {
			i++
			if i == len(timeStep) {
				return 0, fmt.Errorf("time: invalid duration \"%s\"", timeStep)
			}
		}

		//Reads the number from the detected sub-string
		num, err := strconv.ParseInt(timeStep[index:i], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("time: invalid duration \"%s\"", timeStep)
		}

		//After the number, it looks for known units of both 1 and 2 characters
		multiplier, present := unitMap[timeStep[i:i+1]]
		i++
		if !present && i < len(timeStep) {
			multiplier, present = unitMap[timeStep[i-1:i+1]]
			i++
		}
		if !present {
			return 0, fmt.Errorf("time: invalid duration \"%s\"", timeStep)
		}

		//Adds the result to the total duration and points to the next sub-string
		res += num * multiplier

		index = i
	}

	return time.Duration(res), nil
}

//PrintJsonStruct simply prints any given variable to the log
func PrintJsonStruct(v interface{}) {
	jsonOutput, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	log.Println(string(jsonOutput))
}

//WriteJsonStruct simply stores any given variable to a file
func WriteJsonStruct(v interface{}, filename string) {
	jsonOutput, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.Write(jsonOutput)
	if err != nil {
		panic(err)
	}
}
