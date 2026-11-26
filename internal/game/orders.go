// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

// OrderResultsFileName is the name of the file in a turn's folder that holds
// what became of the orders given in the prior turn.
const OrderResultsFileName = "order-results.json"

// OrderResult is what became of one order an empire gave.
type OrderResult struct {
	// EmpireID is the ID of the empire that gave the order.
	EmpireID int `json:"empire_id"`
	// Line is the number of the order's line in the empire's orders file, and
	// Text the line as written.
	Line int    `json:"line"`
	Text string `json:"text"`
	// Accepted is true if the order was carried out. If it is false, Reason
	// says why the order was rejected.
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}
