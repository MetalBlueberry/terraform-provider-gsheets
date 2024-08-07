variable "service_account_credentials" {
  description = "json value with the token obtained from the console"
  type        = string
}

provider "gsheets" {
  service_account_key = var.service_account_credentials
}
