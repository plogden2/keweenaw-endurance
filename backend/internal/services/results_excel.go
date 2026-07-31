package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

// BuildEventResultsWorkbook creates a standings workbook for every active race
// in an event. It uses the same leaderboard calculations as the live results.
func (s *ResultsService) BuildEventResultsWorkbook(eventID uuid.UUID) (data []byte, filename string, err error) {
	var event models.Event
	if err := s.db.First(&event, "id = ?", eventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrEventNotFound
		}
		return nil, "", err
	}

	var races []models.Race
	if err := s.db.Where("event_id = ? AND status != ?", eventID, "cancelled").Order("start_time ASC").Find(&races).Error; err != nil {
		return nil, "", err
	}

	workbook := excelize.NewFile()
	usedSheetNames := make(map[string]struct{})
	sheetCount := 0
	tableCount := 0
	addSheet := func(name string, headers []string, rows [][]interface{}) error {
		name = uniqueSheetName(name, usedSheetNames)
		if sheetCount == 0 {
			if err := workbook.SetSheetName("Sheet1", name); err != nil {
				return err
			}
		} else if _, err := workbook.NewSheet(name); err != nil {
			return err
		}
		sheetCount++

		for col, header := range headers {
			cell, cellErr := excelize.CoordinatesToCellName(col+1, 1)
			if cellErr != nil {
				return cellErr
			}
			if err := workbook.SetCellValue(name, cell, header); err != nil {
				return err
			}
		}
		for row, values := range rows {
			for col, value := range values {
				cell, cellErr := excelize.CoordinatesToCellName(col+1, row+2)
				if cellErr != nil {
					return cellErr
				}
				if err := workbook.SetCellValue(name, cell, value); err != nil {
					return err
				}
			}
		}

		lastRow := len(rows) + 1
		lastCell, cellErr := excelize.CoordinatesToCellName(len(headers), lastRow)
		if cellErr != nil {
			return cellErr
		}
		tableCount++
		return workbook.AddTable(name, &excelize.Table{
			Range:     fmt.Sprintf("A1:%s", lastCell),
			Name:      fmt.Sprintf("ResultsTable%d", tableCount),
			StyleName: "TableStyleMedium2",
		})
	}

	for _, race := range races {
		var participants []models.Participant
		if err := s.db.Preload("Team").Where("race_id = ?", race.ID).Find(&participants).Error; err != nil {
			return nil, "", err
		}
		participantsByID := make(map[uuid.UUID]models.Participant, len(participants))
		for _, participant := range participants {
			participantsByID[participant.ID.UUID()] = participant
		}

		board, err := s.buildOverallLeaderboard(race.ID.UUID(), nil, map[string]CategoryLegendEntry{})
		if err != nil {
			return nil, "", err
		}
		if err := addSheet(
			fmt.Sprintf("%s individual overall", shortRaceName(race.Name)),
			[]string{"Place", "Racer name", "Bib", "Laps", "Age", "Gender", "Team name"},
			individualResultsRows(board, participantsByID),
		); err != nil {
			return nil, "", err
		}

		var categories []models.Category
		if err := s.db.Where("race_id = ?", race.ID).Order("display_order ASC, name ASC").Find(&categories).Error; err != nil {
			return nil, "", err
		}
		for _, category := range categories {
			categoryID := category.ID.UUID()
			categoryBoard, err := s.buildOverallLeaderboard(race.ID.UUID(), &categoryID, map[string]CategoryLegendEntry{})
			if err != nil {
				return nil, "", err
			}
			if len(categoryBoard) == 0 {
				continue
			}
			if err := addSheet(
				fmt.Sprintf("%s %s", shortRaceName(race.Name), category.Name),
				[]string{"Place", "Racer name", "Bib", "Laps", "Age", "Gender", "Team name"},
				individualResultsRows(categoryBoard, participantsByID),
			); err != nil {
				return nil, "", err
			}
		}

		teamBoard, err := s.BuildTeamLeaderboard(race.ID.UUID())
		if err != nil {
			return nil, "", err
		}
		if len(teamBoard) == 0 {
			continue
		}
		rows := make([][]interface{}, 0, len(teamBoard))
		for _, entry := range teamBoard {
			rows = append(rows, []interface{}{entry.Place, entry.Name, entry.AvgLaps, entry.MemberCount})
		}
		if err := addSheet(
			fmt.Sprintf("%s team overall", shortRaceName(race.Name)),
			[]string{"Place", "Team", "Avg laps", "Members"},
			rows,
		); err != nil {
			return nil, "", err
		}
	}

	if sheetCount == 0 {
		if err := addSheet("Results", []string{"Place", "Racer name", "Bib", "Laps", "Age", "Gender", "Team name"}, nil); err != nil {
			return nil, "", err
		}
	}

	buffer, err := workbook.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), fmt.Sprintf("%s-results-%s.xlsx", eventSlug(event.Name), event.EventDate.Format("20060102")), nil
}

func individualResultsRows(board []LiveOverallEntry, participants map[uuid.UUID]models.Participant) [][]interface{} {
	rows := make([][]interface{}, 0, len(board))
	for _, entry := range board {
		participant := participants[entry.ParticipantID.UUID()]
		teamName := ""
		if participant.Team != nil {
			teamName = participant.Team.Name
		}
		rows = append(rows, []interface{}{
			entry.Place,
			entry.Name,
			entry.BibNumber,
			entry.Laps,
			participant.Age,
			participant.Gender,
			teamName,
		})
	}
	return rows
}

func shortRaceName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(
		"-", " ",
		"minute", "min",
		"minutes", "min",
		"hour", "hour",
		"hours", "hour",
	)
	normalized = replacer.Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func uniqueSheetName(name string, used map[string]struct{}) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case ':', '\\', '/', '?', '*', '[', ']':
			return -1
		default:
			return r
		}
	}, name)
	if name == "" {
		name = "Results"
	}
	if len([]rune(name)) > 31 {
		name = string([]rune(name)[:31])
	}
	base := name
	for suffix := 2; ; suffix++ {
		key := strings.ToLower(name)
		if _, exists := used[key]; !exists {
			used[key] = struct{}{}
			return name
		}
		suffixText := fmt.Sprintf(" (%d)", suffix)
		name = truncateSheetName(base, 31-len([]rune(suffixText))) + suffixText
	}
}

func truncateSheetName(name string, limit int) string {
	runes := []rune(name)
	if len(runes) <= limit {
		return name
	}
	return string(runes[:limit])
}

func eventSlug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	slug := strings.Trim(nonSlugCharacters.ReplaceAllString(b.String(), "-"), "-")
	if slug == "" {
		return "event"
	}
	return slug
}
