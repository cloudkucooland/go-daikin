package daikin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Device struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmwareVersion"`
	client			*Client
}

type MSPPayload struct {
	Mode         Mode     `json:"mode"` 
	HeatSetpoint float64 `json:"heatSetpoint"`
	CoolSetpoint float64 `json:"coolSetpoint"`
}

type Info struct {
	EquipmentStatus     int     `json:"equipmentStatus"`
	Mode                int     `json:"mode"`
	ModeLimit           int     `json:"modeLimit"`
	ModeEMHeatAvailable bool    `json:"modeEmHeatAvailable"`
	Fan                 int     `json:"fan"`
	FanCirculate        int     `json:"fanCirculate"`
	FanCirculateSpeed   int     `json:"fanCirculateSpeed"`
	HeatSetpoint        float64 `json:"heatSetpoint"`
	CoolSetpoint        float64 `json:"coolSetpoint"`
	SetPointDelta       int     `json:"setpointDelta"`
	SetPointMinimum     float64 `json:"setpointMinimum"`
	SetPointMaximum     float64 `json:"setpointMaximum"`
	IndoorTemp          float64 `json:"tempIndoor"`
	IndoorHumidity      int     `json:"humIndoor"`
	OutdoorTemp         float64 `json:"tempOutdoor"`
	OutdoorHumidity     int     `json:"humOutdoor"`
	ScheduleEnabled     bool    `json:"scheduleEnabled"`
	GeofencingEnabled   bool    `json:"geofencingEnabled"`
}

func (d *Device) SetTemps(ctx context.Context, mode Mode, heat, cool float64) error {
	url := fmt.Sprintf("/devices/%s/msp", d.ID)

	payload := MSPPayload{
		Mode:         mode,
		HeatSetpoint: heat,
		CoolSetpoint: cool,
	}

	body, _ := json.Marshal(payload)
	res, err := d.client.doRequest(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set auto mode: %s", res.Status)
	}

	slog.Info("Daikin set to deep cool", "device", d.ID, "mode", mode, "cool", cool, "heat", heat)
	return nil
}

func (d *Device) SetModeSchedule(ctx context.Context) error {
	url := fmt.Sprintf("/devices/%s/schedule", d.ID)
	payload := struct {
		ScheduleEnabled bool `json:"scheduleEnabled"`
	}{
		ScheduleEnabled: true,
	}

	body, _ := json.Marshal(payload)
	res, err := d.client.doRequest(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to toggle schedule: %s", res.Status)
	}

	// slog.Info("Daikin set to Schedule", "device", d.ID)
	return nil
}

func (d *Device) GetInfo(ctx context.Context) (*Info, error) {
	url := fmt.Sprintf("/devices/%s", d.ID)
	res, err := d.client.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var info Info
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode info: %w", err)
	}

	// slog.Info("Daikin data pulled", "indoor temp", info.IndoorTemp, "indoor humidity", info.IndoorHumidity)
	return &info, nil
}
