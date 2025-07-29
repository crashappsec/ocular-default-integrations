# Ocular Default Integrations Release Notes
<!-- https://keepachangelog.com -->

# [v0.1.1](https://github.com/crashappsec/ocular/releases/tag/v0.1.1) - **July 29, 2025**

### Added

- **Crawlers**
  - Added a static crawler that supports crawling a static list of target identifiers, given in JSON.

### Fixed

- Improve logging of all default integrations to provide more context and clarity.
- Decrease token cache time to live from 1 hour to 5 minutes to ensure more frequent updates and reduce the risk of using stale tokens.


# [v0.1.0](https://github.com/crashappsec/ocular/releases/tag/v0.1.0) - **July 15, 2025**

### Added

- **Crawlers**
    - GitHub crawler that supports crawling all repositories in a list of organizations.
    - GitLab crawler that supports crawling all repositories in a list of groups, or an entire GitLab instance.
- **Downloaders**
  - git downloader that supports cloning git repositories.
  - docker downloader that supports pulling container images from a registry.
  - NPM downloader that supports downloading NPM packages from a registry.
  - PyPi downloader that supports downloading Python packages from a registry.
  - S3 downloader that supports downloading files from an S3 bucket.
  - GCS downloader that supports downloading files from a Google Cloud Storage bucket.
- **Uploaders**
  - S3 uploader that supports uploading files to an S3 bucket.
  - Webhook uploader that supports uploading files to a webhook endpoint.
