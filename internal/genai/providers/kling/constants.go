package kling

// Task resource kinds for composite task IDs (kling:<kind>:<providerTaskID>).
const (
	TaskText2Video        = "t2v"
	TaskImage2Video       = "i2v"
	TaskMultiImage2Video  = "m2v"
	TaskVideoExtend       = "vext"
	TaskOmniVideo         = "ovid"
	TaskImageGen          = "igen"
	TaskImageExpand       = "iexp"
	TaskOmniImage         = "oimg"
	TaskMultiImage2Image = "m2i"
)

const taskIDPrefix = "kling:"

const defaultBaseURL = "https://api-singapore.klingai.com"
