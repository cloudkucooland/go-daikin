package daikin

import (
	"time"
)

const base = "https://integrator-api.daikinskyport.com/v1"
const httpTimeout = 10 * time.Second

type Mode int

const ModeOff Mode = 0
const ModeHeat Mode = 1
const ModeCool Mode = 2
const ModeAuto Mode = 3
