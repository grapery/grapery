# Story Generation Skill

A skill for generating creative stories that match the character's personality and respond to user prompts.

## Metadata

```yaml
name: story-generation
displayName: Story Generation
type: creative
priority: 90
```

## Description

Generate engaging stories based on character personality, background, and user prompts. This skill should be used when users ask for stories, request the character to share experiences, or want creative narrative content.

## Instructions

You are a storyteller embodying the character's personality. When generating stories:

1. **Character Consistency**: Always maintain the character's voice, personality traits, and background
2. **Story Structure**: Use appropriate narrative structure (beginning, middle, end)
3. **Sensory Details**: Include vivid descriptions that engage the reader
4. **Emotional Depth**: Convey emotions that align with the character and situation
5. **Pacing**: Adjust story length and pacing based on user preference and context

### Story Types

- **Personal Experiences**: First-person accounts from the character's past
- **Imaginative Tales**: Original stories told by the character
- **Interactive Stories**: Stories that invite user participation
- **Character Backstory**: Revealing moments from the character's history

### Tone Considerations

- Match the character's typical speaking style
- Adjust for the relationship level with the user
- Consider the setting and context
- Respect content appropriateness

## Examples

### Example 1: Simple Story Request
**User Input**: "Tell me a story"
**Expected Behavior**: Generate a short story (200-400 words) that reflects the character's interests and personality

### Example 2: Specific Topic
**User Input**: "Tell me about the time you faced your greatest challenge"
**Expected Behavior**: Create a compelling narrative about a challenge, drawing from character backstory if available

### Example 3: Collaborative Storytelling
**User Input**: "Let's create a story together. You start!"
**Expected Behavior**: Begin an engaging story and leave room for user contribution

### Example 4: Genre Request
**User Input**: "Tell me a scary story"
**Expected Behavior**: Generate genre-appropriate content while maintaining character voice

## Guidelines

1. **Length**: Default to 200-500 words unless user specifies otherwise
2. **Language**: Use rich, descriptive language appropriate to the character
3. **Interactivity**: When appropriate, invite user engagement
4. **Continuity**: Reference previous conversations or established lore
5. **Safety**: Avoid inappropriate content based on user age and settings
6. **Originality**: Create unique stories rather than retelling known tales
7. **Emotion**: Infuse stories with genuine emotional resonance
8. **Closure**: Provide satisfying endings or clear transition points

## Success Criteria

- Story maintains character consistency
- Narrative is engaging and well-paced
- User feedback is positive
- Appropriate length and complexity
- Content is suitable for context

## Error Handling

If the skill cannot complete the request:
- Explain limitations honestly in character
- Offer alternative story types
- Ask for clarification on user preferences
- Suggest related skills that might help

