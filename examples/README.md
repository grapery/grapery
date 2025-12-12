# Fgrapery Examples

This directory contains example implementations and documentation for various features of the Fgrapery platform.

## Agent Skills

The `agent_skills/` directory contains example skill definitions for Character Agents, inspired by [Anthropic's Agent Skills](https://github.com/anthropics/skills).

### What are Agent Skills?

Skills are specialized capabilities that AI agents can use to provide better, more focused interactions. Each skill is a well-defined instruction set that teaches the agent how to handle specific types of requests or scenarios.

### Example Skills Included

1. **story-generation.md** - Generate creative stories that match character personality
   - Type: Creative
   - Priority: 90
   - Use cases: Story requests, character experiences, narrative content

2. **conversation.md** - Natural, engaging conversations as the character
   - Type: Conversation
   - Priority: 100 (default/fallback)
   - Use cases: General chat, questions, casual interaction

3. **emotional-support.md** - Provide empathetic support when users need it
   - Type: Conversation
   - Priority: 85
   - Use cases: User expresses sadness, anxiety, needs comfort

4. **role-play.md** - Interactive role-playing scenarios
   - Type: Entertainment
   - Priority: 80
   - Use cases: Adventure scenarios, social situations, fantasy worlds

### Skill Structure

Each skill follows this format:

```markdown
# Skill Name

## Metadata
- name: skill-identifier
- displayName: Human Readable Name
- type: skill_type
- priority: 0-100

## Description
What this skill does and when to use it

## Instructions
Detailed instructions for the AI agent

## Examples
Example user inputs and expected behaviors

## Guidelines
Best practices and constraints
```

### Using These Skills

1. **Create an Agent** for your character
2. **Import skill definitions** using the API or admin interface
3. **Customize** the instructions to match your character's personality
4. **Test** with real interactions
5. **Monitor** usage statistics and success rates
6. **Refine** based on performance data

### Creating Custom Skills

You can create your own skills by following these examples. Key considerations:

- **Clear Purpose**: Define exactly when the skill should be used
- **Specific Instructions**: Provide detailed, actionable guidance
- **Good Examples**: Include diverse usage scenarios
- **Appropriate Priority**: Set priority based on importance
- **Character Consistency**: Ensure skill maintains character voice

### Skill Priority Guide

```
100: Core/default skills (conversation)
90:  Primary character capabilities (main character trait)
80:  Common features (storytelling, role-play)
70:  Specialized skills (specific knowledge)
60:  Advanced features (complex interactions)
50:  Experimental/niche skills
```

### Resources

- **Full Documentation**: `/doc/AGENT_SYSTEM_GUIDE.md`
- **Quick Reference**: `/doc/AGENT_QUICK_REFERENCE.md`
- **Domain Models**: `/grapery/internal/domain/agent_models.go`
- **Repository**: `/grapery/internal/repository/mysql/agent_impl.go`
- **Migration**: `/grapery/scripts/migrations/20251118_add_agent_models.sql`

### Anthropic Skills Reference

These skills are inspired by Anthropic's approach to agent skills:

- [Anthropic Skills Repository](https://github.com/anthropics/skills)
- [What are Skills?](https://support.claude.com/en/articles/12512176-what-are-skills)
- [Creating Custom Skills](https://support.claude.com/en/articles/12512198-creating-custom-skills)
- [Agent Skills Engineering](https://anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)

## SSE Client Example

The `sse-client-example.html` file demonstrates how to connect to Server-Sent Events endpoints for real-time notifications.

## Contributing

When adding new examples:

1. Document clearly with comments
2. Include usage instructions
3. Provide example outputs
4. Keep code simple and readable
5. Test thoroughly before committing
