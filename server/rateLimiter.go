package server

import (
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

var (
	D_MAX        = 15               // nolint
	D_EXPIRATION = 30 * time.Second // nolint
)

func rateLimiterConfig() *limiter.Config {
	updateDefaultValues()

	config := limiter.Config{
		Max:        D_MAX, // {Max} request in per {Expiration} interval
		Expiration: D_EXPIRATION,
	}

	return &config
}

func updateDefaultValues() {
	maxLimit := os.Getenv("RL_MAX_LIMIT")
	if maxLimit != "" {
		maxLimitGiven, err := strconv.Atoi(maxLimit)
		if err != nil {
			log.Error("Failed to parse RL_MAX_LIMIT value, using default", "maxLimit", D_MAX)
		} else {
			D_MAX = maxLimitGiven
			log.Info("Using max limit", "max_limit", D_MAX)
		}
	} else {
		log.Info("Using default max limit value", "max_limit", D_MAX)
	}

	expTime := os.Getenv("RL_EXPIRATION")
	if expTime != "" {
		exp, err := time.ParseDuration(expTime)
		if err != nil {
			log.Error("Failed to parse RL_EXPIRATION value, using default", "expTime", D_EXPIRATION)
		} else {
			D_EXPIRATION = exp
			log.Info("Using Expiration time", "exp_time", D_EXPIRATION)
		}
	} else {
		log.Info("Using default expiration value", "exp_time", D_EXPIRATION)
	}
}
