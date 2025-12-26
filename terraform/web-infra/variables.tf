variable "aws_region" {
    description = "AWS 리전"
    default = "ap-northeast-2"
}

variable "bucket_name" {
  description = "프론트 서빙 S3 버킷"
  type = string
}