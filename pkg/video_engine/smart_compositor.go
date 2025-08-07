package video_engine

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"creative-studio-server/models"
	"creative-studio-server/pkg/logger"
)

type SmartCompositor struct {
	clips         []models.AtomicClip
	requirements  CompositionRequirements
	algorithms    map[string]CompositionAlgorithm
}

type CompositionRequirements struct {
	TargetDuration    float64   `json:"target_duration"`
	Theme             string    `json:"theme"`
	Mood              string    `json:"mood"`
	Style             string    `json:"style"`
	PrimaryColors     []string  `json:"primary_colors"`
	SecondaryColors   []string  `json:"secondary_colors"`
	MusicTempo        string    `json:"music_tempo"` // slow, medium, fast
	TransitionStyle   string    `json:"transition_style"`
	MinClipDuration   float64   `json:"min_clip_duration"`
	MaxClipDuration   float64   `json:"max_clip_duration"`
	ContentBalance    map[string]float64 `json:"content_balance"` // e.g., {"close_up": 0.3, "wide_shot": 0.4, "medium_shot": 0.3}
	AvoidRepetition   bool      `json:"avoid_repetition"`
	PreferHighQuality bool      `json:"prefer_high_quality"`
}

type CompositionResult struct {
	SelectedClips     []ClipSegment     `json:"selected_clips"`
	Timeline          []TimelineEvent   `json:"timeline"`
	TotalDuration     float64           `json:"total_duration"`
	QualityScore      float64           `json:"quality_score"`
	CohesionScore     float64           `json:"cohesion_score"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type ClipSegment struct {
	ClipID       uint    `json:"clip_id"`
	StartTime    float64 `json:"start_time"`
	EndTime      float64 `json:"end_time"`
	Duration     float64 `json:"duration"`
	Score        float64 `json:"score"`
	Reason       string  `json:"reason"`
	Transitions  []Transition `json:"transitions"`
}

type TimelineEvent struct {
	Type       string      `json:"type"` // clip, transition, effect
	StartTime  float64     `json:"start_time"`
	Duration   float64     `json:"duration"`
	Properties interface{} `json:"properties"`
}

type Transition struct {
	Type     string  `json:"type"`
	Duration float64 `json:"duration"`
	Easing   string  `json:"easing"`
}

type CompositionAlgorithm interface {
	Score(clip models.AtomicClip, requirements CompositionRequirements, context CompositionContext) float64
	SelectClips(clips []models.AtomicClip, requirements CompositionRequirements) ([]ClipSegment, error)
}

type CompositionContext struct {
	PreviousClips    []models.AtomicClip
	CurrentPosition  float64
	RemainingTime    float64
}

func NewSmartCompositor(clips []models.AtomicClip, requirements CompositionRequirements) *SmartCompositor {
	compositor := &SmartCompositor{
		clips:        clips,
		requirements: requirements,
		algorithms:   make(map[string]CompositionAlgorithm),
	}

	// Register composition algorithms
	compositor.algorithms["smart_selection"] = &SmartSelectionAlgorithm{}
	compositor.algorithms["theme_based"] = &ThemeBasedAlgorithm{}
	compositor.algorithms["emotion_driven"] = &EmotionDrivenAlgorithm{}

	return compositor
}

func (sc *SmartCompositor) GenerateComposition(ctx context.Context, algorithmName string) (*CompositionResult, error) {
	logger.Infof("Starting smart composition generation with algorithm: %s", algorithmName)

	algorithm, exists := sc.algorithms[algorithmName]
	if !exists {
		algorithm = sc.algorithms["smart_selection"] // Default algorithm
	}

	// Score and filter clips
	scoredClips := sc.scoreClips(algorithm)
	
	// Select clips based on algorithm
	selectedClips, err := algorithm.SelectClips(scoredClips, sc.requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to select clips: %w", err)
	}

	// Generate timeline
	timeline := sc.generateTimeline(selectedClips)

	// Calculate scores
	qualityScore := sc.calculateQualityScore(selectedClips)
	cohesionScore := sc.calculateCohesionScore(selectedClips)

	result := &CompositionResult{
		SelectedClips: selectedClips,
		Timeline:      timeline,
		TotalDuration: sc.calculateTotalDuration(timeline),
		QualityScore:  qualityScore,
		CohesionScore: cohesionScore,
		Metadata: map[string]interface{}{
			"algorithm":       algorithmName,
			"clip_count":      len(selectedClips),
			"generation_time": time.Now(),
		},
	}

	logger.Infof("Composition generated successfully: %d clips, %.2fs duration, quality: %.2f, cohesion: %.2f",
		len(selectedClips), result.TotalDuration, qualityScore, cohesionScore)

	return result, nil
}

func (sc *SmartCompositor) scoreClips(algorithm CompositionAlgorithm) []models.AtomicClip {
	scored := make([]models.AtomicClip, len(sc.clips))
	copy(scored, sc.clips)

	// Score each clip
	for i := range scored {
		context := CompositionContext{
			PreviousClips:   []models.AtomicClip{},
			CurrentPosition: 0,
			RemainingTime:   sc.requirements.TargetDuration,
		}
		
		score := algorithm.Score(scored[i], sc.requirements, context)
		// Store score in metadata (you might want to add a Score field to AtomicClip)
		if scored[i].Metadata == nil {
			scored[i].Metadata = make(models.JSON)
		}
		scored[i].Metadata["composition_score"] = score
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		scoreI, _ := scored[i].Metadata["composition_score"].(float64)
		scoreJ, _ := scored[j].Metadata["composition_score"].(float64)
		return scoreI > scoreJ
	})

	return scored
}

func (sc *SmartCompositor) generateTimeline(clips []ClipSegment) []TimelineEvent {
	var timeline []TimelineEvent
	currentTime := 0.0

	for i, clip := range clips {
		// Add clip event
		timeline = append(timeline, TimelineEvent{
			Type:      "clip",
			StartTime: currentTime,
			Duration:  clip.Duration,
			Properties: map[string]interface{}{
				"clip_id":    clip.ClipID,
				"start_time": clip.StartTime,
				"end_time":   clip.EndTime,
			},
		})

		currentTime += clip.Duration

		// Add transition if not the last clip
		if i < len(clips)-1 {
			transition := sc.selectTransition(clip, clips[i+1])
			timeline = append(timeline, TimelineEvent{
				Type:      "transition",
				StartTime: currentTime - transition.Duration/2,
				Duration:  transition.Duration,
				Properties: transition,
			})
		}
	}

	return timeline
}

func (sc *SmartCompositor) selectTransition(fromClip, toClip ClipSegment) Transition {
	// Intelligent transition selection based on clip properties
	transitionTypes := []string{"fade", "dissolve", "slide", "wipe", "cut"}
	
	// Default transition
	selectedType := "dissolve"
	duration := 0.5

	// Adjust based on transition style requirement
	switch sc.requirements.TransitionStyle {
	case "fast":
		selectedType = "cut"
		duration = 0.1
	case "smooth":
		selectedType = "dissolve"
		duration = 1.0
	case "dynamic":
		selectedType = transitionTypes[rand.Intn(len(transitionTypes))]
		duration = 0.3
	}

	return Transition{
		Type:     selectedType,
		Duration: duration,
		Easing:   "ease-in-out",
	}
}

func (sc *SmartCompositor) calculateQualityScore(clips []ClipSegment) float64 {
	if len(clips) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, clip := range clips {
		totalScore += clip.Score
	}

	return totalScore / float64(len(clips))
}

func (sc *SmartCompositor) calculateCohesionScore(clips []ClipSegment) float64 {
	if len(clips) <= 1 {
		return 1.0
	}

	// Calculate how well clips flow together
	// This is a simplified implementation
	cohesionScore := 0.0
	comparisons := 0

	for i := 0; i < len(clips)-1; i++ {
		// Compare adjacent clips for coherence
		// In a real implementation, you'd analyze visual similarity,
		// color continuity, motion consistency, etc.
		similarity := sc.calculateClipSimilarity(clips[i], clips[i+1])
		cohesionScore += similarity
		comparisons++
	}

	if comparisons == 0 {
		return 1.0
	}

	return cohesionScore / float64(comparisons)
}

func (sc *SmartCompositor) calculateClipSimilarity(clip1, clip2 ClipSegment) float64 {
	// Simplified similarity calculation
	// In practice, this would involve deep analysis of visual features
	similarity := 0.5 // Base similarity

	// This would be enhanced with actual video analysis
	return similarity
}

func (sc *SmartCompositor) calculateTotalDuration(timeline []TimelineEvent) float64 {
	if len(timeline) == 0 {
		return 0.0
	}

	maxEndTime := 0.0
	for _, event := range timeline {
		endTime := event.StartTime + event.Duration
		if endTime > maxEndTime {
			maxEndTime = endTime
		}
	}

	return maxEndTime
}

// Smart Selection Algorithm Implementation
type SmartSelectionAlgorithm struct{}

func (a *SmartSelectionAlgorithm) Score(clip models.AtomicClip, requirements CompositionRequirements, context CompositionContext) float64 {
	score := 0.0

	// Duration fitness (prefer clips that fit well)
	durationFitness := a.calculateDurationFitness(clip.Duration, requirements)
	score += durationFitness * 0.3

	// Theme/mood matching
	themeFitness := a.calculateThemeFitness(clip, requirements)
	score += themeFitness * 0.4

	// Quality score (resolution, bitrate, etc.)
	qualityFitness := a.calculateQualityFitness(clip, requirements)
	score += qualityFitness * 0.3

	return score
}

func (a *SmartSelectionAlgorithm) SelectClips(clips []models.AtomicClip, requirements CompositionRequirements) ([]ClipSegment, error) {
	var selectedClips []ClipSegment
	remainingDuration := requirements.TargetDuration
	usedClips := make(map[uint]bool)

	for remainingDuration > requirements.MinClipDuration && len(selectedClips) < len(clips) {
		bestClip := a.findBestClip(clips, usedClips, remainingDuration, requirements)
		if bestClip == nil {
			break
		}

		clipDuration := bestClip.Duration
		if clipDuration > remainingDuration {
			clipDuration = remainingDuration
		}

		selectedClips = append(selectedClips, ClipSegment{
			ClipID:    bestClip.ID,
			StartTime: 0,
			EndTime:   clipDuration,
			Duration:  clipDuration,
			Score:     bestClip.Metadata["composition_score"].(float64),
			Reason:    "Smart selection algorithm",
		})

		usedClips[bestClip.ID] = true
		remainingDuration -= clipDuration
	}

	return selectedClips, nil
}

func (a *SmartSelectionAlgorithm) findBestClip(clips []models.AtomicClip, usedClips map[uint]bool, remainingDuration float64, requirements CompositionRequirements) *models.AtomicClip {
	for _, clip := range clips {
		if usedClips[clip.ID] {
			continue
		}
		
		if clip.Duration >= requirements.MinClipDuration {
			return &clip
		}
	}
	return nil
}

func (a *SmartSelectionAlgorithm) calculateDurationFitness(duration float64, requirements CompositionRequirements) float64 {
	if duration < requirements.MinClipDuration || duration > requirements.MaxClipDuration {
		return 0.0
	}
	
	ideal := (requirements.MinClipDuration + requirements.MaxClipDuration) / 2
	deviation := abs(duration - ideal)
	maxDeviation := requirements.MaxClipDuration - ideal
	
	return 1.0 - (deviation / maxDeviation)
}

func (a *SmartSelectionAlgorithm) calculateThemeFitness(clip models.AtomicClip, requirements CompositionRequirements) float64 {
	fitness := 0.0
	
	if requirements.Theme != "" && clip.Category == requirements.Theme {
		fitness += 0.5
	}
	
	if requirements.Mood != "" && clip.Mood == requirements.Mood {
		fitness += 0.3
	}
	
	if requirements.Style != "" && clip.Style == requirements.Style {
		fitness += 0.2
	}
	
	return fitness
}

func (a *SmartSelectionAlgorithm) calculateQualityFitness(clip models.AtomicClip, requirements CompositionRequirements) float64 {
	fitness := 0.0
	
	// Resolution quality
	if clip.Resolution == "1920x1080" {
		fitness += 0.5
	} else if clip.Resolution == "1280x720" {
		fitness += 0.3
	}
	
	// Bitrate quality
	if clip.Bitrate >= 2000 {
		fitness += 0.3
	} else if clip.Bitrate >= 1000 {
		fitness += 0.2
	}
	
	// Frame rate smoothness
	if clip.FrameRate >= 30 {
		fitness += 0.2
	}
	
	return fitness
}

// Theme-based Algorithm (placeholder implementations)
type ThemeBasedAlgorithm struct{}

func (a *ThemeBasedAlgorithm) Score(clip models.AtomicClip, requirements CompositionRequirements, context CompositionContext) float64 {
	// Implementation would focus heavily on theme coherence
	return rand.Float64()
}

func (a *ThemeBasedAlgorithm) SelectClips(clips []models.AtomicClip, requirements CompositionRequirements) ([]ClipSegment, error) {
	// Simplified implementation
	return []ClipSegment{}, nil
}

// Emotion-driven Algorithm (placeholder implementations)
type EmotionDrivenAlgorithm struct{}

func (a *EmotionDrivenAlgorithm) Score(clip models.AtomicClip, requirements CompositionRequirements, context CompositionContext) float64 {
	// Implementation would analyze emotional flow and pacing
	return rand.Float64()
}

func (a *EmotionDrivenAlgorithm) SelectClips(clips []models.AtomicClip, requirements CompositionRequirements) ([]ClipSegment, error) {
	// Simplified implementation
	return []ClipSegment{}, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}