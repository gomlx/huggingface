package hub

import (
	"fmt"
	"github.com/gomlx/gomlx/ml/data/downloader"
	"github.com/pkg/errors"
	"log"
	"os"
	"path"
	"strings"
)

// Repo from which one wants to download files. Create it with New.
type Repo struct {
	// ID of the Repo may include owner/model. E.g.: google/gemma-2-2b-it
	ID string

	// repoType of the repository, usually RepoTypeModel.
	repoType RepoType

	// revision to download, usually set to "main", but it can use a commit-hash version.
	revision string

	// authToken is the HuggingFace authentication token to be used when downloading the files.
	authToken string

	// Verbosity: 0 for quiet operation; 1 for information about progress; 2 and higher for debugging.
	Verbosity int

	// MaxParallelDownload indicates how many files to download at the same time. Default is 20.
	// If set to <= 0 it will download all files in parallel.
	// Set to 1 to make downloads sequential.
	MaxParallelDownload int

	// cacheDir is where to store the downloaded files.
	cacheDir string

	// Info about the Repo in HuggingFace, including the list of files.
	// It is only available after DownloadInfo is called.
	info *RepoInfo

	downloadManager *downloader.Manager

	useProgressBar bool
}

// New creates a reference to a HuggingFace model given its id.
//
// The id typically include owner/model. E.g.: "google/gemma-2-2b-it"
//
// It defaults to being a RepoTypeModel repository. But you can change it with Repo.WithType.
//
// If authentication is needed, use Repo.WithAuth.
func New(id string) *Repo {
	return &Repo{
		ID:                  id,
		repoType:            RepoTypeModel,
		revision:            "main",
		cacheDir:            DefaultCacheDir(),
		Verbosity:           1,
		MaxParallelDownload: 20, // At most 20 parallel downloads.
	}
}

// WithAuth sets the authentication token to use during downloads.
func (r *Repo) WithAuth(authToken string) *Repo {
	r.authToken = authToken
	return r
}

// WithType sets the repository type to use during downloads.
func (r *Repo) WithType(repoType RepoType) *Repo {
	r.repoType = repoType
	return r
}

// WithRevision sets the revision to use for this Repo, defaults to "main", but can be set to a commit-hash value.
func (r *Repo) WithRevision(repoType RepoType) *Repo {
	r.repoType = repoType
	return r
}

// WithCacheDir sets the cacheDir to the given directory.
//
// The default is given by DefaultCacheDir: `${XDG_CACHE_HOME}/huggingface/hub` if set, or `~/.cache/huggingface/hub` otherwise.
func (r *Repo) WithCacheDir(cacheDir string) *Repo {
	newCacheDir, err := ReplaceTildeInDir(cacheDir)
	if err == nil {
		r.cacheDir = path.Clean(newCacheDir)
	} else {
		log.Printf("Failed to resolve directory for %q: %+v", cacheDir, err)
	}
	return r
}

// WithDownloadManager sets the downloader.Manager to use for download.
// This is not needed, one will be created automatically if one is not set.
// This is useful when downloading multiple Repos simultaneously, to coordinate limits by sharing the download manager.
func (r *Repo) WithDownloadManager(manager *downloader.Manager) *Repo {
	r.downloadManager = manager
	return r
}

// WithProgressBar configures the usage of progress bar during download. Defaults to true.
func (r *Repo) WithProgressBar(useProgressBar bool) *Repo {
	r.useProgressBar = useProgressBar
	return r
}

// flatFolderName returns a serialized version of a hf.co repo name and type, safe for disk storage
// as a single non-nested folder.
//
// Based on github.com/huggingface/huggingface_hub repo_folder_name.
func (r *Repo) flatFolderName() string {
	parts := []string{string(r.repoType)}
	parts = append(parts, strings.Split(r.ID, "/")...)
	return strings.Join(parts, RepoIdSeparator)
}

// repoCache joins cacheDir and flatFolderName to return the cache sub-directory for the repository.
// It also creates the directory, and returns an error if creation failed.
func (r *Repo) repoCache() (string, error) {
	dir := path.Join(r.cacheDir, r.flatFolderName())
	err := os.MkdirAll(dir, DefaultDirCreationPerm)
	if err != nil {
		return "", errors.Wrapf(err, "while creating cache directory %q", dir)
	}
	return dir, nil
}

// FileURL returns the URL from which to download the file from HuggingFace.
//
// Usually, not used directly (use DownloadFile instead), but in case someone needs for debugging.
func (r *Repo) FileURL(fileName string) string {
	return fmt.Sprintf("https://huggingface.co/%s/%s/resolve/%s/%s",
		r.repoType, r.ID, r.revision, fileName)
}

// readCommitHashForRevision finds the commit-hash for the revision, it should already be written to disk.
// The revision can be itself a commit-hash, in which case it is returned directly.
//
// repoCacheDir is returned by Repo.repoCache().
func (r *Repo) readCommitHashForRevision() (string, error) {
	err := r.DownloadInfo(false)
	if err != nil {
		return "", err
	}
	return r.info.CommitHash, nil
}