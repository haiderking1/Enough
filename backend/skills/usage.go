package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type UsageRecord struct {
	CreatedBy     *string `json:"created_by"`
	UseCount      int     `json:"use_count"`
	ViewCount     int     `json:"view_count"`
	LastUsedAt    *string `json:"last_used_at"`
	LastViewedAt  *string `json:"last_viewed_at"`
	PatchCount    int     `json:"patch_count"`
	LastPatchedAt *string `json:"last_patched_at"`
	CreatedAt     string  `json:"created_at"`
	State         string  `json:"state"`
	Pinned        bool    `json:"pinned"`
	ArchivedAt    *string `json:"archived_at"`
}

type UsageMap map[string]UsageRecord

type UsageReportRow struct {
	Name           string      `json:"name"`
	CreatedBy      *string     `json:"created_by"`
	UseCount       int         `json:"use_count"`
	ViewCount      int         `json:"view_count"`
	LastUsedAt     *string     `json:"last_used_at"`
	LastViewedAt   *string     `json:"last_viewed_at"`
	PatchCount     int         `json:"patch_count"`
	LastPatchedAt  *string     `json:"last_patched_at"`
	CreatedAt      string      `json:"created_at"`
	State          string      `json:"state"`
	Pinned         bool        `json:"pinned"`
	ArchivedAt     *string     `json:"archived_at"`
	LastActivityAt *string     `json:"last_activity_at"`
	ActivityCount  int         `json:"activity_count"`
}

var usageMu sync.Mutex

func usageFilePath() string {
	return filepath.Join(SkillsDir(), ".usage.json")
}

func nowIso() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func emptyRecord() UsageRecord {
	return UsageRecord{
		CreatedBy:     nil,
		UseCount:      0,
		ViewCount:     0,
		LastUsedAt:    nil,
		LastViewedAt:  nil,
		PatchCount:    0,
		LastPatchedAt: nil,
		CreatedAt:     nowIso(),
		State:         "active",
		Pinned:        false,
		ArchivedAt:    nil,
	}
}

func parseIso(value *string) (time.Time, bool) {
	if value == nil || *value == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, *value)
	if err == nil {
		return t, true
	}
	t, err = time.Parse(time.RFC3339Nano, *value)
	if err == nil {
		return t, true
	}
	return time.Time{}, false
}

func LatestActivityAt(record UsageRecord) *string {
	var latestTime time.Time
	var latestRaw *string

	fields := []*string{record.LastUsedAt, record.LastViewedAt, record.LastPatchedAt}
	for _, raw := range fields {
		if t, ok := parseIso(raw); ok {
			if t.After(latestTime) {
				latestTime = t
				latestRaw = raw
			}
		}
	}
	return latestRaw
}

func ActivityCount(record UsageRecord) int {
	return record.UseCount + record.ViewCount + record.PatchCount
}

func LoadUsage() UsageMap {
	path := usageFilePath()
	dataBytes, err := os.ReadFile(path)
	if err != nil {
		return make(UsageMap)
	}
	var um UsageMap
	if err := json.Unmarshal(dataBytes, &um); err != nil {
		return make(UsageMap)
	}
	return um
}

func SaveUsage(um UsageMap) {
	path := usageFilePath()
	keys := make([]string, 0, len(um))
	for k := range um {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sorted := make(map[string]UsageRecord)
	for _, k := range keys {
		sorted[k] = um[k]
	}

	dataBytes, err := json.MarshalIndent(sorted, "", "  ")
	if err != nil {
		return
	}
	_ = atomicWrite(path, dataBytes)
}

func atomicWrite(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, "enough-atomic-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filename)
}

func mutateUsage(name string, f func(*UsageRecord)) {
	if name == "" {
		return
	}
	usageMu.Lock()
	defer usageMu.Unlock()

	um := LoadUsage()
	rec, ok := um[name]
	if !ok {
		rec = emptyRecord()
	}
	f(&rec)
	um[name] = rec
	SaveUsage(um)
}

func BumpView(name string) {
	mutateUsage(name, func(rec *UsageRecord) {
		rec.ViewCount++
		t := nowIso()
		rec.LastViewedAt = &t
	})
}

func BumpUse(name string) {
	mutateUsage(name, func(rec *UsageRecord) {
		rec.UseCount++
		t := nowIso()
		rec.LastUsedAt = &t
	})
}

func BumpPatch(name string) {
	mutateUsage(name, func(rec *UsageRecord) {
		rec.PatchCount++
		t := nowIso()
		rec.LastPatchedAt = &t
	})
}

func MarkAgentCreated(name string) {
	mutateUsage(name, func(rec *UsageRecord) {
		a := "agent"
		rec.CreatedBy = &a
	})
}

func Forget(name string) {
	if name == "" {
		return
	}
	usageMu.Lock()
	defer usageMu.Unlock()

	um := LoadUsage()
	if _, ok := um[name]; ok {
		delete(um, name)
		SaveUsage(um)
	}
}

func SetState(name, state string) {
	valid := map[string]bool{"active": true, "stale": true, "archived": true}
	if !valid[state] {
		return
	}
	mutateUsage(name, func(rec *UsageRecord) {
		rec.State = state
		if state == "archived" {
			t := nowIso()
			rec.ArchivedAt = &t
		} else if state == "active" {
			rec.ArchivedAt = nil
		}
	})
}

func SetPinned(name string, pinned bool) {
	mutateUsage(name, func(rec *UsageRecord) {
		rec.Pinned = pinned
	})
}

func PinSkill(name string) {
	SetPinned(name, true)
}

func UnpinSkill(name string) {
	SetPinned(name, false)
}

func findSkillDir(name string) string {
	skillsRoot := SkillsDir()
	if _, err := os.Stat(skillsRoot); os.IsNotExist(err) {
		return ""
	}
	var foundDir string
	_ = filepath.Walk(skillsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if info.Name() == ".archive" {
			return filepath.SkipDir
		}
		if info.Name() == name {
			if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err == nil {
				foundDir = path
				return filepath.SkipAll
			}
		}
		return nil
	})
	return foundDir
}

func ListAgentCreatedSkillNames() []string {
	um := LoadUsage()
	var names []string
	for k, v := range um {
		if v.CreatedBy != nil && *v.CreatedBy == "agent" {
			if findSkillDir(k) != "" {
				names = append(names, k)
			}
		}
	}
	sort.Strings(names)
	return names
}

func AgentCreatedReport() []UsageReportRow {
	um := LoadUsage()
	var rows []UsageReportRow
	for _, name := range ListAgentCreatedSkillNames() {
		rec := um[name]
		rows = append(rows, UsageReportRow{
			Name:           name,
			CreatedBy:      rec.CreatedBy,
			UseCount:       rec.UseCount,
			ViewCount:      rec.ViewCount,
			LastUsedAt:     rec.LastUsedAt,
			LastViewedAt:   rec.LastViewedAt,
			PatchCount:     rec.PatchCount,
			LastPatchedAt:  rec.LastPatchedAt,
			CreatedAt:      rec.CreatedAt,
			State:          rec.State,
			Pinned:         rec.Pinned,
			ArchivedAt:     rec.ArchivedAt,
			LastActivityAt: LatestActivityAt(rec),
			ActivityCount:  ActivityCount(rec),
		})
	}
	return rows
}

func ListArchivedSkillNames() []string {
	root := ArchiveDir()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

func ArchiveSkill(name string) (bool, string) {
	skillDir := findSkillDir(name)
	if skillDir == "" {
		return false, fmt.Sprintf("skill '%s' not found", name)
	}
	root := ArchiveDir()
	if err := os.MkdirAll(root, 0o700); err != nil {
		return false, fmt.Sprintf("failed to create archive dir: %v", err)
	}
	dest := filepath.Join(root, name)
	if _, err := os.Stat(dest); err == nil {
		ts := time.Now().Format("20060102150405")
		dest = filepath.Join(root, fmt.Sprintf("%s-%s", name, ts))
	}
	if err := os.Rename(skillDir, dest); err != nil {
		return false, fmt.Sprintf("failed to archive: %v", err)
	}
	SetState(name, "archived")
	return true, fmt.Sprintf("archived to %s", dest)
}

func RestoreSkill(name string) (bool, string) {
	root := ArchiveDir()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return false, "no archive directory"
	}
	var src string
	exact := filepath.Join(root, name)
	if fi, err := os.Stat(exact); err == nil && fi.IsDir() {
		src = exact
	} else {
		archived := ListArchivedSkillNames()
		var newest string
		for _, n := range archived {
			if strings.HasPrefix(n, name+"-") {
				if n > newest {
					newest = n
				}
			}
		}
		if newest != "" {
			src = filepath.Join(root, newest)
		}
	}
	if src == "" {
		return false, fmt.Sprintf("skill '%s' not found in archive", name)
	}
	dest := filepath.Join(SkillsDir(), name)
	if _, err := os.Stat(dest); err == nil {
		return false, fmt.Sprintf("destination already exists: %s", dest)
	}
	if err := os.Rename(src, dest); err != nil {
		return false, fmt.Sprintf("failed to restore: %v", err)
	}
	SetState(name, "active")
	return true, fmt.Sprintf("restored to %s", dest)
}
