package documents

// Direct upload and completion discover these capabilities through interfaces.
// Assert the complete contracts here so a signature change cannot silently
// disable uploads in one implementation while another implementation compiles.
var _ ObjectStore = (*MemoryObjectStore)(nil)
var _ UploadSigner = (*MemoryObjectStore)(nil)
var _ ObjectMetadataReader = (*MemoryObjectStore)(nil)
var _ ObjectContentReader = (*MemoryObjectStore)(nil)

var _ ObjectStore = (*S3ObjectStore)(nil)
var _ UploadSigner = (*S3ObjectStore)(nil)
var _ ObjectMetadataReader = (*S3ObjectStore)(nil)
var _ ObjectContentReader = (*S3ObjectStore)(nil)
