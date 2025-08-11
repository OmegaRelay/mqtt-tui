package updater

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"time"
)

type Version struct {
	major int
	minor int
	patch int
}

type Updater struct {
	repo   string
	curVer Version

	// Cache the latest asset
	latestAsset *GithubAsset
}

type GithubAssets []GithubAsset
type GithubAsset struct {
	URL      string `json:"url"`
	ID       int    `json:"id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	Label    string `json:"label"`
	Uploader struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		UserViewType      string `json:"user_view_type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"uploader"`
	ContentType        string    `json:"content_type"`
	State              string    `json:"state"`
	Size               int       `json:"size"`
	Digest             string    `json:"digest"`
	DownloadCount      int       `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadURL string    `json:"browser_download_url"`
}

type GithubReleases []GithubRelease
type GithubRelease struct {
	URL       string `json:"url"`
	AssetsURL string `json:"assets_url"`
	UploadURL string `json:"upload_url"`
	HTMLURL   string `json:"html_url"`
	ID        int    `json:"id"`
	Author    struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		UserViewType      string `json:"user_view_type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	NodeID          string       `json:"node_id"`
	TagName         string       `json:"tag_name"`
	TargetCommitish string       `json:"target_commitish"`
	Name            string       `json:"name"`
	Draft           bool         `json:"draft"`
	Immutable       bool         `json:"immutable"`
	Prerelease      bool         `json:"prerelease"`
	CreatedAt       time.Time    `json:"created_at"`
	PublishedAt     time.Time    `json:"published_at"`
	Assets          GithubAssets `json:"assets"`
	TarballURL      string       `json:"tarball_url"`
	ZipballURL      string       `json:"zipball_url"`
	Body            string       `json:"body"`
}

func New(repo string, currentVersion Version) Updater {
	return Updater{
		repo:   repo,
		curVer: currentVersion,
	}
}

func (u *Updater) IsUpdateAvailable() (bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", u.repo)
	rsp, err := http.DefaultClient.Get(url)
	if err != nil {
		return false, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(rsp.Body)
	releases := GithubReleases{}
	err = json.Unmarshal(buf.Bytes(), &releases)
	if err != nil {
		return false, nil
	}

	var newVer Version
	var latestRelease *GithubRelease
	for _, release := range releases {
		ver, err := parseVersionTag(release.TagName)
		if err != nil {
			continue
		}
		if u.curVer.IsGreaterThan(ver) || newVer.IsGreaterThan(ver) {
			continue
		}
		newVer = ver
		latestRelease = &release
	}

	if u.curVer.IsGreaterThan(newVer) {
		return false, nil
	}

	asset := latestRelease.Assets.MatchesOS()
	if asset == nil {
		return false, nil
	}

	u.latestAsset = asset

	return true, nil
}

func (u *Updater) InstallLatestUpdate() error {
	if u.latestAsset == nil {
		isUpdateAvailable, err := u.IsUpdateAvailable()
		if err != nil {
			return err
		}
		if !isUpdateAvailable {
			return errors.New("no new update is available")
		}
	}

	rsp, err := http.DefaultClient.Get(u.latestAsset.BrowserDownloadURL)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(rsp.Body)
	tmpDir, err := os.MkdirTemp("", "updater-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(u.latestAsset.Name, buf.Bytes(), 0700)
	if err != nil {
		return err
	}

	return nil
}

func parseVersionTag(tag string) (Version, error) {
	ver := Version{}
	_, err := fmt.Sscanf(tag, "v%d.%d.%d", &ver.major, &ver.minor, &ver.patch)
	if err != nil {
		return Version{}, err
	}

	return ver, nil
}

func (v Version) IsGreaterThan(n Version) bool {
	return v.major > n.major || v.minor > n.minor || v.patch > n.patch
}

func (a GithubAssets) MatchesOS() *GithubAsset {
	r, _ := regexp.Compile("(" + runtime.GOOS + "[-_]" + runtime.GOARCH + ")+")
	for i, asset := range a {
		if r.MatchString(asset.Label) || r.MatchString(asset.Name) {
			return &a[i]
		}
	}
	return nil
}
