package model

// Feature names a capability or modality supported by a model.
type Feature string

const (
	// CapChat marks conversational chat support.
	CapChat Feature = "CapChat"
	// CapFunctionCall marks native tool / function calling support.
	CapFunctionCall Feature = "CapFunctionCall"
	// CapJsonMode marks structured JSON output mode support.
	CapJsonMode Feature = "CapJsonMode"
	// ModalityTextIn marks text input modality support.
	ModalityTextIn Feature = "ModalityTextIn"
	// ModalityTextOut marks text output modality support.
	ModalityTextOut Feature = "ModalityTextOut"
	// ModalityImageIn marks image input modality support.
	ModalityImageIn Feature = "ModalityImageIn"
	// ModalityAudioIn marks audio input modality support.
	ModalityAudioIn Feature = "ModalityAudioIn"
	// ModalityVideoIn marks video input modality support.
	ModalityVideoIn Feature = "ModalityVideoIn"
	// ModalityFileIn marks file input modality support.
	ModalityFileIn Feature = "ModalityFileIn"
)
