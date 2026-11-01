package main

import "npc-gui/internal/model"

// BinaryResult is returned by binary detection and download actions.
type BinaryResult struct {
	Binary     *model.BinaryInfo     `json:"binary,omitempty"`
	Candidates []model.BinaryInfo    `json:"candidates"`
	Result     model.OperationResult `json:"result"`
}

// ProfileSaveResult is returned when a profile is saved.
type ProfileSaveResult struct {
	Profile        model.Profile         `json:"profile"`
	PendingRestart bool                  `json:"pendingRestart"`
	Result         model.OperationResult `json:"result"`
}

// ProfilesSaveResult is returned for bulk profile creation/import.
type ProfilesSaveResult struct {
	Profiles []model.Profile       `json:"profiles"`
	Result   model.OperationResult `json:"result"`
}

// SettingsSaveResult wraps saved settings.
type SettingsSaveResult struct {
	Settings model.Settings        `json:"settings"`
	Result   model.OperationResult `json:"result"`
}

// StringResult wraps a path or message.
type StringResult struct {
	Value  string                `json:"value,omitempty"`
	Result model.OperationResult `json:"result"`
}

// UpdateStateResult wraps an updater snapshot.
type UpdateStateResult struct {
	State  model.UpdateStateInfo `json:"state"`
	Result model.OperationResult `json:"result"`
}

// ProfileExport is the on-disk import/export envelope.
type ProfileExport struct {
	SchemaVersion  int             `json:"schemaVersion"`
	ExportedAt     int64           `json:"exportedAt"`
	IncludeSecrets bool            `json:"includeSecrets"`
	Profiles       []model.Profile `json:"profiles"`
}

// UpdateActionResult is returned when an update is applied.
type UpdateActionResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}
