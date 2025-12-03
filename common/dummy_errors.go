package common

// This is just here so we don't get an error about these error codes
// not being used. Once we add support for the corresponding features, and
// the error checks, we should delete them from here.

func dummy_errors() {
	NewXRError("compatibility_violation", "/")
	NewXRError("data_retrieval_error", "/")
	NewXRError("model_compliance_error", "/")
	NewXRError("too_large", "/")
}
