package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"gorm.io/gorm"
)

const (
	MinTeamMembers = 2
	MaxTeamMembers = 12
)

var (
	ErrTeamNotFound     = errors.New("team not found")
	ErrInvalidTeamInput = errors.New("invalid team input")
)

type TeamService struct {
	db *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{db: db}
}

func (s *TeamService) ListTeamsByRace(raceID uuid.UUID) ([]models.Team, error) {
	var teams []models.Team
	if err := s.db.Preload("Participants").Where("race_id = ?", raceID).
		Order("display_order ASC, name ASC").Find(&teams).Error; err != nil {
		return nil, err
	}
	return teams, nil
}

func (s *TeamService) GetTeam(id uuid.UUID) (*models.Team, error) {
	var team models.Team
	if err := s.db.Preload("Participants").First(&team, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return &team, nil
}

func (s *TeamService) CreateTeam(input *models.Team) (*models.Team, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: team is required", ErrInvalidTeamInput)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidTeamInput)
	}
	if input.RaceID.IsZero() {
		return nil, fmt.Errorf("%w: race_id is required", ErrInvalidTeamInput)
	}
	if err := s.ensureRaceExists(input.RaceID.UUID()); err != nil {
		return nil, err
	}

	team := models.Team{
		RaceID:       input.RaceID,
		Name:         name,
		DisplayOrder: input.DisplayOrder,
	}
	if err := s.db.Create(&team).Error; err != nil {
		return nil, err
	}
	return &team, nil
}

// UpdateTeamFields applies optional name/display_order patches.
func (s *TeamService) UpdateTeamFields(id uuid.UUID, name *string, displayOrder *int) (*models.Team, error) {
	team, err := s.GetTeam(id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: name is required", ErrInvalidTeamInput)
		}
		team.Name = trimmed
	}
	if displayOrder != nil {
		team.DisplayOrder = *displayOrder
	}
	if err := s.db.Save(team).Error; err != nil {
		return nil, err
	}
	return team, nil
}

func (s *TeamService) DeleteTeam(id uuid.UUID) error {
	team, err := s.GetTeam(id)
	if err != nil {
		return err
	}
	if err := s.db.Model(&models.Participant{}).
		Where("team_id = ?", team.ID).
		Update("team_id", nil).Error; err != nil {
		return err
	}
	result := s.db.Delete(&models.Team{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTeamNotFound
	}
	return nil
}

// SetMembers replaces the team's roster. Empty clears all. Exactly 1 member is rejected.
func (s *TeamService) SetMembers(teamID uuid.UUID, participantIDs []uuid.UUID) (*models.Team, error) {
	team, err := s.GetTeam(teamID)
	if err != nil {
		return nil, err
	}

	n := len(participantIDs)
	if n == 1 {
		return nil, fmt.Errorf("%w: team requires at least %d members (or zero to clear)", ErrInvalidTeamInput, MinTeamMembers)
	}
	if n > MaxTeamMembers {
		return nil, fmt.Errorf("%w: team may have at most %d members", ErrInvalidTeamInput, MaxTeamMembers)
	}

	seen := map[uuid.UUID]struct{}{}
	for _, id := range participantIDs {
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("%w: duplicate participant_id", ErrInvalidTeamInput)
		}
		seen[id] = struct{}{}
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Participant{}).
			Where("team_id = ?", team.ID).
			Update("team_id", nil).Error; err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		var members []models.Participant
		if err := tx.Where("id IN ?", participantIDs).Find(&members).Error; err != nil {
			return err
		}
		if len(members) != n {
			return fmt.Errorf("%w: one or more participants not found", ErrInvalidTeamInput)
		}
		for _, m := range members {
			if m.RaceID != team.RaceID {
				return fmt.Errorf("%w: participant must belong to the same race", ErrInvalidTeamInput)
			}
		}
		teamIDPub := team.ID
		if err := tx.Model(&models.Participant{}).
			Where("id IN ?", participantIDs).
			Update("team_id", teamIDPub).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetTeam(teamID)
}

func (s *TeamService) ensureRaceExists(raceID uuid.UUID) error {
	var race models.Race
	if err := s.db.First(&race, "id = ?", raceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: race not found", ErrInvalidTeamInput)
		}
		return err
	}
	return nil
}

func (s *TeamService) ensureTeamOnRace(teamID, raceID uuid.UUID) error {
	var team models.Team
	if err := s.db.First(&team, "id = ?", teamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: team not found", ErrInvalidParticipantInput)
		}
		return err
	}
	if team.RaceID.UUID() != raceID {
		return fmt.Errorf("%w: team must belong to the same race", ErrInvalidParticipantInput)
	}
	return nil
}

func (s *TeamService) ValidateTeamForRace(teamID, raceID uuid.UUID) error {
	return s.ensureTeamOnRace(teamID, raceID)
}
