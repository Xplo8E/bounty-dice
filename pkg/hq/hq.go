package hq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/schollz/progressbar/v3"
)

const (
	ME_QUERY = `query MeQuery {
  me {
    id
  }
}`

	PROGRAM_HANDLES_QUERY = `query DiscoveryQuery($query: OpportunitiesQuery!, $filter: QueryInput!, $from: Int, $size: Int, $sort: [SortInput!], $post_filters: OpportunitiesFilterInput) {
  opportunities_search(
    query: $query
    filter: $filter
    from: $from
    size: $size
    sort: $sort
    post_filters: $post_filters
  ) {
    nodes {
      ... on OpportunityDocument {
        handle
      }
    }
    total_count
  }
}
`
	TEAMPROFILE_QUERY = `query TeamProfile($handle: String!) {
  team(handle: $handle) {
    ...BountyTable
    ...ProfileMetrics
  }
}

fragment BountyTable on Team {
  profile_metrics_snapshot {
    average_bounty_per_severity_low
    average_bounty_per_severity_medium
    average_bounty_per_severity_high
    average_bounty_per_severity_critical
    report_count_per_severity_low
    report_count_per_severity_medium
    report_count_per_severity_high
    report_count_per_severity_critical
  }
  bounty_table {
    low_label
    medium_label
    high_label
    critical_label
    description
    use_range
    bounty_table_rows(first: 100) {
      nodes {
        low
        medium
        high
        critical
        low_minimum
        medium_minimum
        high_minimum
        critical_minimum
        smart_rewards_start_at
        structured_scope {
          id
          asset_identifier
        }
        updated_at
      }
    }
    updated_at
  }
}

fragment ProfileMetrics on Team {
  currency
  offers_bounties
  average_bounty_lower_amount
  average_bounty_upper_amount
  top_bounty_lower_amount
  top_bounty_upper_amount
  formatted_total_bounties_paid_prefix
  formatted_total_bounties_paid_amount
  resolved_report_count
  formatted_bounties_paid_last_90_days
  hide_bounty_amounts
  reports_received_last_90_days
  last_report_resolved_at
  most_recent_sla_snapshot {
    first_response_time: average_time_to_first_program_response
    triage_time: average_time_to_report_triage
    bounty_time: average_time_to_bounty_awarded
    resolution_time: average_time_to_report_resolved
  }
  team_display_options {
    show_response_efficiency_indicator
    show_mean_first_response_time
    show_mean_report_triage_time
    show_mean_bounty_time
    show_mean_resolution_time
    show_top_bounties
    show_average_bounty
    show_total_bounties_paid
  }
}`
)

var verboseLog bool

func SetVerbose(v bool) {
	verboseLog = v
}

func verbose(format string, a ...interface{}) {
	if verboseLog {
		fmt.Printf("[VERBOSE] "+format+"\n", a...)
	}
}

type Session struct {
	HostSessionCookie string
	CSRFToken         string
	Client            *http.Client
}

type HQProgram struct {
	Handle   string
	Findings []string
}

func NewSession(hostSessionCookie, csrfToken string) *Session {
	verbose("Creating new HQ session.")
	return &Session{
		HostSessionCookie: hostSessionCookie,
		CSRFToken:         csrfToken,
		Client:            &http.Client{},
	}
}

func (s *Session) post(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://hackerone.com/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://hackerone.com")
	req.Header.Set("x-csrf-token", s.CSRFToken)
	req.AddCookie(&http.Cookie{Name: "__Host-session", Value: s.HostSessionCookie})

	verbose("Posting GraphQL query to HackerOne.")
	resp, err := s.Client.Do(req)
	if err != nil {
		verbose("Error during GraphQL request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("non-200 status code returned (%d): %s", resp.StatusCode, string(body))
		verbose("GraphQL request failed: %v", err)
		return nil, err
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		verbose("Error decoding GraphQL response: %v", err)
		return nil, err
	}

	verbose("GraphQL request successful.")
	return result, nil
}

func (s *Session) CheckAuth() bool {
	verbose("Checking HackerOne authentication status.")
	res, err := s.post(ME_QUERY, map[string]interface{}{})
	if err != nil {
		verbose("Auth check failed during post: %v", err)
		return false
	}

	data, ok := res["data"].(map[string]interface{})
	if !ok {
		verbose("Auth check failed: 'data' field not found.")
		return false
	}

	me, ok := data["me"].(map[string]interface{})
	if !ok {
		verbose("Auth check failed: 'me' field not found.")
		return false
	}

	_, ok = me["id"]
	if ok {
		verbose("Authentication successful.")
	} else {
		verbose("Authentication failed: 'id' not found in 'me'.")
	}
	return ok
}

func (s *Session) GetProgramInfo(handle string) (map[string]interface{}, error) {
	verbose("Getting program info for handle: %s", handle)
	variables := map[string]interface{}{
		"handle": handle,
	}
	return s.post(TEAMPROFILE_QUERY, variables)
}

func CheckProgramData(handle string, minReq int, tData map[string]interface{}) (bool, []string) {
	verbose("Checking program data for handle: %s (minReq: %d)", handle, minReq)
	teamData := tData

	bountyTableRows, ok := teamData["bounty_table"].(map[string]interface{})
	if !ok {
		verbose("[%s] 'bounty_table' not found or not a map.", handle)
		return false, nil
	}
	nodes, ok := bountyTableRows["bounty_table_rows"].(map[string]interface{})
	if !ok {
		verbose("[%s] 'bounty_table_rows' not found or not a map.", handle)
		return false, nil
	}
	btr, ok := nodes["nodes"].([]interface{})
	if !ok || len(btr) == 0 {
		verbose("[%s] 'nodes' not found, not an array, or is empty.", handle)
		return false, nil
	}

	maxPayout, _ := teamData["top_bounty_upper_amount"].(float64)
	var maxBountyCrit float64
	for _, bountyTable := range btr {
		bt, ok := bountyTable.(map[string]interface{})
		if !ok {
			continue
		}
		crit, _ := bt["critical"].(float64)
		critMin, _ := bt["critical_minimum"].(float64)
		maxBountyCrit = max(maxBountyCrit, max(crit, critMin))
	}

	var maxBountyHigh float64
	for _, bountyTable := range btr {
		bt, ok := bountyTable.(map[string]interface{})
		if !ok {
			continue
		}
		high, _ := bt["high"].(float64)
		highMin, _ := bt["high_minimum"].(float64)
		maxBountyHigh = max(maxBountyHigh, max(high, highMin))
	}
	verbose("[%s] Max Payout: $%.2f, Max Crit Bounty: $%.2f, Max High Bounty: $%.2f", handle, maxPayout, maxBountyCrit, maxBountyHigh)

	var findings []string
	if metricsSnapshot, ok := teamData["profile_metrics_snapshot"].(map[string]interface{}); ok && metricsSnapshot != nil {
		if maxBountyHigh > 0 {
			if avgBountyHigh, ok := metricsSnapshot["average_bounty_per_severity_high"].(float64); ok {
				avgHighPercent := avgBountyHigh / maxBountyHigh
				if avgHighPercent >= 0.75 {
					finding := fmt.Sprintf("Avg high payout is >= 75%% of max high ($%.0f vs $%.0f)", avgBountyHigh, maxBountyHigh)
					findings = append(findings, finding)
					verbose("[%s] Finding: %s", handle, finding)
				}
			}
		}

		if maxBountyCrit > 0 {
			if avgBountyCrit, ok := metricsSnapshot["average_bounty_per_severity_critical"].(float64); ok {
				if avgCritPercent := avgBountyCrit / maxBountyCrit; avgCritPercent >= 0.75 {
					finding := fmt.Sprintf("Avg crit payout is >= 75%% of max crit ($%.0f vs $%.0f)", avgBountyCrit, maxBountyCrit)
					findings = append(findings, finding)
					verbose("[%s] Finding: %s", handle, finding)
				}
			}
		}

		lowReportCount, _ := metricsSnapshot["report_count_per_severity_low"].(float64)
		mediumReportCount, _ := metricsSnapshot["report_count_per_severity_medium"].(float64)
		highReportCount, _ := metricsSnapshot["report_count_per_severity_high"].(float64)
		criticalReportCount, _ := metricsSnapshot["report_count_per_severity_critical"].(float64)
		totalReportMetricsCount := lowReportCount + mediumReportCount + highReportCount + criticalReportCount

		if totalReportMetricsCount > 0 {
			if highReportPercent := highReportCount / totalReportMetricsCount; highReportPercent >= 0.4 {
				finding := fmt.Sprintf("High-severity reports make up %.0f%% of total", highReportPercent*100)
				findings = append(findings, finding)
				verbose("[%s] Finding: %s", handle, finding)
			}
			if criticalReportPercent := criticalReportCount / totalReportMetricsCount; criticalReportPercent >= 0.4 {
				finding := fmt.Sprintf("Crit-severity reports make up %.0f%% of total", criticalReportPercent*100)
				findings = append(findings, finding)
				verbose("[%s] Finding: %s", handle, finding)
			}
			if highAndCritReportPercent := (highReportCount + criticalReportCount) / totalReportMetricsCount; highAndCritReportPercent >= 0.4 {
				finding := fmt.Sprintf("High+Crit reports make up %.0f%% of total", highAndCritReportPercent*100)
				findings = append(findings, finding)
				verbose("[%s] Finding: %s", handle, finding)
			}
		}
	}

	if maxPayout > 0 && maxBountyCrit > 0 && maxPayout > maxBountyCrit {
		finding := fmt.Sprintf("Max bounty ($%.0f) > max crit bounty ($%.0f)", maxPayout, maxBountyCrit)
		findings = append(findings, finding)
		verbose("[%s] Finding: %s", handle, finding)
	}

	isHq := len(findings) >= minReq
	verbose("[%s] Total findings: %d. Program is HQ: %t", handle, len(findings), isHq)
	if isHq {
		return true, findings
	}
	return false, nil
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func FetchAndCheck(session *Session, handles []string, force bool, minReq int) ([]HQProgram, error) {
	var allProgData map[string]interface{}
	dataPath := "~/.cache/bounty-dice/program_data.json"
	if _, err := os.Stat(dataPath); os.IsNotExist(err) || force {
		if force {
			verbose("Force flag is set. Re-fetching all program data.")
		} else {
			verbose("'%s' not found. Fetching new program data.", dataPath)
		}

		if !session.CheckAuth() {
			fmt.Println("[!] WARNING: Accessing HackerOne API unauthenticated. Only public programs will be fetched!")
		}

		fmt.Printf("Got %d program handles, fetching and saving program data now...\n", len(handles))
		var bar *progressbar.ProgressBar
		if !verboseLog {
			bar = progressbar.Default(int64(len(handles)))
		}

		allProgData = make(map[string]interface{})
		for _, handle := range handles {
			if bar != nil {
				bar.Add(1)
			}
			pInfo, err := session.GetProgramInfo(handle)
			if err != nil {
				verbose("[!] Uhoh, null team data returned for program: %s (error: %v)", handle, err)
				continue
			}
			if teamData, ok := pInfo["data"].(map[string]interface{}); ok {
				if team, ok := teamData["team"]; ok {
					allProgData[handle] = team
				} else {
					verbose("[!] 'team' key not found in data for handle: %s", handle)
				}
			} else {
				verbose("[!] 'data' key not found in response for handle: %s", handle)
			}
		}

		verbose("Finished fetching data for all handles. Marshalling and saving to '%s'.", dataPath)
		data, err := json.MarshalIndent(allProgData, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshalling program data: %w", err)
		}
		os.WriteFile(dataPath, data, 0644)
		verbose("Successfully saved program data to '%s'.", dataPath)

	} else {
		verbose("Loading program data from '%s'.", dataPath)
		data, err := os.ReadFile(dataPath)
		if err != nil {
			return nil, fmt.Errorf("error reading ~/.cache/bounty-dice/program_data.json: %w", err)
		}
		json.Unmarshal(data, &allProgData)
		verbose("Successfully loaded %d programs from cache.", len(allProgData))
	}

	var hqPrograms []HQProgram
	verbose("Checking %d cached programs against HQ criteria.", len(allProgData))
	for handle, teamData := range allProgData {
		if teamData == nil {
			verbose("Skipping nil team data for handle: %s", handle)
			continue
		}
		if isHq, findings := CheckProgramData(handle, minReq, teamData.(map[string]interface{})); isHq {
			hqPrograms = append(hqPrograms, HQProgram{Handle: handle, Findings: findings})
		}
	}
	verbose("Found %d HQ programs after checking.", len(hqPrograms))
	return hqPrograms, nil
}
