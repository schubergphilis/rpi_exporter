/*
Package prometheus provides utilities for authoring a Prometheus metric exporter for the metrics
found in the Mailbox Property Interface of a Raspberry Pi.
*/
package prometheus

// NB: We could implement a Prometheus Collector, but this simple exporter allows us to avoid
// dependencies. The exposition format is formalized here:
//
// https://prometheus.io/docs/instrumenting/exposition_formats/

import (
	"fmt"
	"io"

	"github.com/schubergphilis/rpi_exporter/pkg/mbox"
)

const (
	metricTypeGauge = "gauge"
)

var voltageLabelsByID = map[mbox.VoltageID]string{
	mbox.VoltageIDCore:   "core",
	mbox.VoltageIDSDRAMC: "sdram_c",
	mbox.VoltageIDSDRAMI: "sdram_i",
	mbox.VoltageIDSDRAMP: "sdram_p",
}

var powerLabelsByID = map[mbox.PowerDeviceID]string{
	mbox.PowerDeviceIDSDCard: "sd_card",
	mbox.PowerDeviceIDUART0:  "uart0",
	mbox.PowerDeviceIDUART1:  "uart1",
	mbox.PowerDeviceIDUSBHCD: "usb_hcd",
	mbox.PowerDeviceIDI2C0:   "i2c0",
	mbox.PowerDeviceIDI2C1:   "i2c1",
	mbox.PowerDeviceIDI2C2:   "i2c2",
	mbox.PowerDeviceIDSPI:    "spi",
	mbox.PowerDeviceIDCCP2TX: "ccp2tx",
}

var clockLabelsByID = map[mbox.ClockID]string{
	mbox.ClockIDEMMC:     "emmc",
	mbox.ClockIDUART:     "uart",
	mbox.ClockIDARM:      "arm",
	mbox.ClockIDCore:     "core",
	mbox.ClockIDV3D:      "v3d",
	mbox.ClockIDH264:     "h264",
	mbox.ClockIDISP:      "isp",
	mbox.ClockIDSDRAM:    "sdram",
	mbox.ClockIDPixel:    "pixel",
	mbox.ClockIDPWM:      "pwm",
	mbox.ClockIDHEVC:     "hevc",
	mbox.ClockIDEMMC2:    "emmc2",
	mbox.ClockIDM2MC:     "m2mc",
	mbox.ClockIDPixelBVB: "pixel_bvb",
}

func formatTemp(t float32) string  { return fmt.Sprintf("%.03f", t) }
func formatVolts(v float32) string { return fmt.Sprintf("%.06f", v) }

func formatBool(b bool) string {
	if b {
		return "1"
	}

	return "0"
}

type expWriter struct {
	w      io.Writer
	name   string
	labels []string
}

// Write all metrics in Prometheus text-based exposition format.
func Write(w io.Writer) error {
	ew := &expWriter{w: w}

	return ew.write()
}

func (w *expWriter) writeHeaderGauge(name, help string, labels ...string) {
	w.name = name
	w.labels = labels
	fmt.Fprintf(w.w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w.w, "# TYPE %s %v\n", name, "gauge")
}

func (w *expWriter) writeSample(val interface{}, labels ...string) {
	if len(labels) != len(w.labels) {
		panic("developer error: incorrect metrics label count")
	}

	fmt.Fprint(w.w, w.name)

	if len(w.labels) > 0 {
		fmt.Fprintf(w.w, "{")

		for i, key := range w.labels {
			if i > 0 {
				fmt.Fprintf(w.w, ",")
			}

			fmt.Fprintf(w.w, "%s=\"%s\"", key, labels[i])
		}

		fmt.Fprintf(w.w, "}")
	}

	fmt.Fprintf(w.w, " %v\n", val)
}

func (w *expWriter) write() error {
	mboxOpen, err := mbox.Open()
	if err != nil {
		return fmt.Errorf("unable to open mbox: %w", err)
	}

	defer mboxOpen.Close()

	if err := w.writeHardware(mboxOpen); err != nil {
		return err
	}

	if err := w.writePower(mboxOpen); err != nil {
		return err
	}

	if err := w.writeClocks(mboxOpen); err != nil {
		return err
	}

	if err := w.writeTemperatures(mboxOpen); err != nil {
		return err
	}

	if err := w.writeVoltages(mboxOpen); err != nil {
		return err
	}

	return nil
}

func (w *expWriter) writeHardware(mboxOpen *mbox.Mailbox) error {
	w.writeHeaderGauge("rpi_vc_revision", "Firmware revision of the VideoCore device.", metricTypeGauge)

	rev, err := mboxOpen.GetFirmwareRevision()
	if err != nil {
		return fmt.Errorf("unable to get firmware revision: %w", err)
	}

	w.writeSample(rev)

	w.writeHeaderGauge("rpi_board_model", "Board model.", metricTypeGauge)

	model, err := mboxOpen.GetBoardModel()
	if err != nil {
		return fmt.Errorf("unable to get board model: %w", err)
	}

	w.writeSample(model)

	w.writeHeaderGauge("rpi_board_revision", "Board revision.", metricTypeGauge)

	rev, err = mboxOpen.GetBoardRevision()
	if err != nil {
		return fmt.Errorf("unable to get board revision: %w", err)
	}

	w.writeSample(rev)

	return nil
}

func (w *expWriter) writePower(mboxOpen *mbox.Mailbox) error {
	w.writeHeaderGauge("rpi_power_state", "Component power state (0: off, 1: on, 2: missing).", metricTypeGauge, "id")

	for id, label := range powerLabelsByID {
		state, err := mboxOpen.GetPowerState(id)
		if err != nil {
			return fmt.Errorf("unable to get power state: %w", err)
		}

		w.writeSample(state, label)
	}

	return nil
}

func (w *expWriter) writeClocks(mboxOpen *mbox.Mailbox) error {
	w.writeHeaderGauge("rpi_clock_rate_hz", "Clock rate in Hertz.", metricTypeGauge, "id")

	for id, label := range clockLabelsByID {
		rate, err := mboxOpen.GetClockRate(id)
		if err != nil {
			return fmt.Errorf("unable to get clock rate: %w", err)
		}

		w.writeSample(rate, label)
	}

	w.writeHeaderGauge("rpi_clock_rate_measured_hz", "Measured clock rate in Hertz.", metricTypeGauge, "id")

	for id, label := range clockLabelsByID {
		rate, err := mboxOpen.GetClockRateMeasured(id)
		if err != nil {
			return fmt.Errorf("unable to get measured clock rate: %w", err)
		}

		w.writeSample(rate, label)
	}

	w.writeHeaderGauge("rpi_turbo", "Turbo state.", metricTypeGauge)

	turbo, err := mboxOpen.GetTurbo()
	if err != nil {
		return fmt.Errorf("unable to get turbo: %w", err)
	}

	w.writeSample(formatBool(turbo))

	return nil
}

func (w *expWriter) writeTemperatures(mboxOpen *mbox.Mailbox) error {
	w.writeHeaderGauge("rpi_temperature_c", "Temperature of the SoC in degrees celsius.", metricTypeGauge, "id")

	temp, err := mboxOpen.GetTemperature()
	if err != nil {
		return fmt.Errorf("unable to get temperature: %w", err)
	}

	w.writeSample(formatTemp(temp), "soc")

	w.writeHeaderGauge("rpi_temperature_f", "Temperature of the SoC in degrees fahrenheit.", metricTypeGauge, "id")
	w.writeSample(formatTemp(temp*9/5+32), "soc")

	w.writeHeaderGauge(
		"rpi_max_temperature_c",
		"Maximum temperature of the SoC in degrees celsius.",
		metricTypeGauge,
		"id",
	)

	maxTemp, err := mboxOpen.GetMaxTemperature()
	if err != nil {
		return fmt.Errorf("unable to get maximum temperature: %w", err)
	}

	w.writeSample(formatTemp(maxTemp), "soc")

	w.writeHeaderGauge(
		"rpi_max_temperature_f",
		"Maximum temperature of the SoC in degrees fahrenheit.",
		metricTypeGauge,
		"id")
	w.writeSample(formatTemp(maxTemp*9/5+32), "soc")

	return nil
}

func (w *expWriter) writeVoltages(mboxOpen *mbox.Mailbox) error {
	w.writeHeaderGauge("rpi_voltage", "Current component voltage.", metricTypeGauge, "id")

	for id, label := range voltageLabelsByID {
		volts, err := mboxOpen.GetVoltage(id)
		if err != nil {
			return fmt.Errorf("unable to get voltage: %w", err)
		}

		w.writeSample(formatVolts(volts), label)
	}

	w.writeHeaderGauge("rpi_voltage_min", "Minimum supported component voltage.", metricTypeGauge, "id")

	for id, label := range voltageLabelsByID {
		volts, err := mboxOpen.GetMinVoltage(id)
		if err != nil {
			return fmt.Errorf("unable to get minimum voltage: %w", err)
		}

		w.writeSample(formatVolts(volts), label)
	}

	w.writeHeaderGauge("rpi_voltage_max", "Maximum supported component voltage.", metricTypeGauge, "id")

	for id, label := range voltageLabelsByID {
		volts, err := mboxOpen.GetMaxVoltage(id)
		if err != nil {
			return fmt.Errorf("unable to get max voltage: %w", err)
		}

		w.writeSample(formatVolts(volts), label)
	}

	return nil
}
