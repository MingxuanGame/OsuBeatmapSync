package utils

import (
	"context"
	"strconv"
	"time"
)

const retryAfterKey string = "Retry-After"

func getRetryAfterFromContext(ctx context.Context) time.Duration {
	if retryAfter, ok := ctx.Value(retryAfterKey).(int); ok {
		return time.Second * time.Duration(retryAfter)
	}
	return 0
}

func GetLimitSecond(sleep string, ctx context.Context) (time.Duration, context.Context, error) {
	sleepNum, err := strconv.Atoi(sleep)
	if err != nil {
		return 0, ctx, err
	}
	retryAfter := getRetryAfterFromContext(ctx)
	if retryAfter > 0 {
		if sleepNum > int(retryAfter.Seconds()) {
			ctx = context.WithValue(ctx, retryAfterKey, sleepNum)
			retryAfter = time.Second * time.Duration(sleepNum)
		}
	} else {
		retryAfter = time.Second * time.Duration(sleepNum)
	}
	return retryAfter, ctx, nil
}
