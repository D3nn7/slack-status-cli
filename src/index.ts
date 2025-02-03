// src/index.ts
import inquirer from 'inquirer';
import chalk from 'chalk';
import { WebClient } from '@slack/web-api';
import fs from 'fs';
import path from 'path';

const TEMPLATES_FILE = path.join(__dirname, '../templates.json');
const CONFIG_FILE = path.join(__dirname, '../config.json');

interface StatusTemplate {
    text: string;
    emoji: string;
    label: string;
    durationInMinutes?: number;
    untilTime?: string;
}

interface Config {
    slackToken: string;
}

function loadConfig(): Config {
    try {
        if (!fs.existsSync(CONFIG_FILE)) {
            console.error(chalk.red('❌ Fehler: config.json nicht gefunden!'));
            console.log(chalk.yellow('Bitte erstelle eine config.json mit folgendem Format:'));
            console.log(chalk.blue(JSON.stringify({ slackToken: "xoxp-dein-slack-token" }, null, 2)));
            process.exit(1);
        }
        const data = fs.readFileSync(CONFIG_FILE, 'utf8');
        const config = JSON.parse(data);
        
        if (!config.slackToken) {
            console.error(chalk.red('❌ Fehler: Slack Token nicht in config.json gefunden!'));
            process.exit(1);
        }
        
        return config;
    } catch (error) {
        console.error(chalk.red('Fehler beim Laden der Konfiguration:', error));
        process.exit(1);
    }
}

const config = loadConfig();
const slack = new WebClient(config.slackToken);

function loadTemplates(): StatusTemplate[] {
    try {
        if (!fs.existsSync(TEMPLATES_FILE)) {
            console.error(chalk.red('❌ Fehler: templates.json nicht gefunden!'));
            process.exit(1);
        }
        const data = fs.readFileSync(TEMPLATES_FILE, 'utf8');
        const jsonData = JSON.parse(data);
        // Stelle sicher, dass wir das templates-Array aus der JSON erhalten
        return jsonData.templates || [];
    } catch (error) {
        console.error(chalk.red('Fehler beim Laden der Templates:', error));
        process.exit(1);
    }
}

function saveTemplates(templates: StatusTemplate[]): void {
    try {
        // Speichere die Templates in der korrekten Struktur
        const jsonData = { templates: templates };
        fs.writeFileSync(TEMPLATES_FILE, JSON.stringify(jsonData, null, 2));
        console.log(chalk.green('✓ Templates erfolgreich gespeichert'));
    } catch (error) {
        console.error(chalk.red('Fehler beim Speichern der Templates:', error));
    }
}

async function clearConsole() {
    process.stdout.write(process.platform === 'win32' ? '\x1B[2J\x1B[0f' : '\x1B[2J\x1B[3J\x1B[H');
}

async function getCurrentStatus(): Promise<string> {
    try {
        const currentUserStatus = await slack.users.profile.get({
            user: undefined
        });
        const currentUser = currentUserStatus?.profile?.display_name;
        const currentText = currentUserStatus?.profile?.status_text || 'Kein Status gesetzt';
        const currentEmoji = currentUserStatus?.profile?.status_emoji || '';
        
        return `${chalk.blue(`👤 Angemeldeter User: ${currentUser}`)}
${chalk.green(`🟢 Aktueller Status: ${currentText} ${currentEmoji}`)}`;
    } catch (error) {
        return chalk.red('Fehler beim Abrufen des Status');
    }
}

async function setSlackStatus(text: string, emoji: string, durationInMinutes?: number, untilTime?: string) {

    let expiration;

    if (durationInMinutes != null) {
        const currentDate = new Date();
        const futureDate = new Date(currentDate.getTime() + durationInMinutes * 60 * 1000);
        expiration = Math.floor(futureDate.getTime() / 1000);
    }

    if (untilTime != null) {
        const timeRegex = /^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/;
        if (timeRegex.test(untilTime)) {
            const [hours, minutes] = untilTime.split(':').map(Number);
            const currentDate = new Date();
            
            currentDate.setHours(hours);
            currentDate.setMinutes(minutes);
            currentDate.setSeconds(0);
            currentDate.setMilliseconds(0);
            
            expiration = Math.floor(currentDate.getTime() / 1000);
        }
    }

    try {
        await slack.users.profile.set({
            profile: {
                status_text: text,
                status_emoji: emoji,
                status_expiration: expiration
            }
        });
        
        console.log(chalk.green('✓ Status erfolgreich aktualisiert!'));
    } catch (error) {
        console.error(chalk.red('Fehler beim Aktualisieren des Status:', error));
    }
}

async function startApplication() {
    await showMainMenu();
}

// Modifiziere alle Prompt-Funktionen, um den Shortcut-Hinweis anzuzeigen
async function showPromptHeader() {
    await clearConsole();
    console.log(await getCurrentStatus());
    console.log('\n' + chalk.blue.bold('🎯 Slack Status Manager (by Danny Schapeit)'));
}

async function showMainMenu() {
    await showPromptHeader();

    const { action } = await inquirer.prompt([
        {
            type: 'list',
            name: 'action',
            message: chalk.blue('Was möchtest du tun?'),
            choices: [
                { name: '📝 Status manuell setzen', value: 'manual' },
                { name: '📋 Template verwenden', value: 'template' },
                { name: '⚡ Status anpassen', value: 'modify' },
                { name: '➕ Neues Template erstellen', value: 'create' },
                { name: '🗑️ Template löschen', value: 'delete' },
                { name: '❌ Beenden', value: 'exit' }
            ]
        }
    ]);

    switch (action) {
        case 'manual':
            await setManualStatus();
            break;
        case 'template':
            await useTemplate();
            break;
        case 'modify':
            await modifyCurrentStatus();
            break;
        case 'create':
            await createTemplate();
            break;
        case 'delete':
            await deleteTemplate();
            break;
        case 'exit':
            process.exit(0);
    }
}

async function modifyCurrentStatus() {
    await clearConsole();
    try {
        const currentStatus = await slack.users.profile.get({});
        const profile = currentStatus.profile;
        
        const { action } = await inquirer.prompt([
            {
                type: 'list',
                name: 'action',
                message: 'Status anpassen:',
                choices: [
                    { name: '✏️  Status bearbeiten', value: 'edit' },
                    new inquirer.Separator(),
                    { name: '↩️  Zurück zum Hauptmenü', value: 'back' }
                ]
            }
        ]);

        if (action === 'back') {
            return showMainMenu();
        }

        const { text, emoji } = await inquirer.prompt([
            {
                type: 'input',
                name: 'text',
                message: 'Neuer Status Text:',
                default: profile?.status_text
            },
            {
                type: 'input',
                name: 'emoji',
                message: 'Neues Emoji:',
                default: profile?.status_emoji
            }
        ]);

        await setSlackStatus(text, emoji);
    } catch (error) {
        console.error(chalk.red('Fehler beim Anpassen des Status:', error));
    }
    showMainMenu();
}

async function setManualStatus() {
    await clearConsole();
    const { action } = await inquirer.prompt([
        {
            type: 'list',
            name: 'action',
            message: 'Status setzen:',
            choices: [
                { name: '✏️  Neuen Status eingeben', value: 'new' },
                new inquirer.Separator(),
                { name: '↩️  Zurück zum Hauptmenü', value: 'back' }
            ]
        }
    ]);

    if (action === 'back') {
        return showMainMenu();
    }

    const { text, emoji } = await inquirer.prompt([
        {
            type: 'input',
            name: 'text',
            message: 'Status Text:',
            validate: (input: string) => input.length > 0 || 'Status Text ist erforderlich'
        },
        {
            type: 'input',
            name: 'emoji',
            message: 'Emoji (z.B. :coffee:):',
            validate: (input: string) => input.length > 0 || 'Emoji ist erforderlich'
        }
    ]);

    await setSlackStatus(text, emoji);
    showMainMenu();
}

async function useTemplate() {
    const templates = loadTemplates();
    
    const { template } = await inquirer.prompt([
        {
            type: 'list',
            name: 'template',
            message: 'Wähle ein Template:',
            choices: [
                ...templates.map(t => ({
                    name: t.label,
                    value: t
                })),
                new inquirer.Separator(),
                { name: '↩️  Zurück zum Hauptmenü', value: 'back' }
            ]
        }
    ]);

    if (template === 'back') {
        return showMainMenu();
    }

    await setSlackStatus(template.text, template.emoji, template.durationInMinutes, template.untilTime);
    showMainMenu();
}

async function createTemplate() {
    await clearConsole();
    const { action } = await inquirer.prompt([
        {
            type: 'list',
            name: 'action',
            message: 'Template erstellen:',
            choices: [
                { name: '➕ Neues Template erstellen', value: 'new' },
                new inquirer.Separator(),
                { name: '↩️  Zurück zum Hauptmenü', value: 'back' }
            ]
        }
    ]);

    if (action === 'back') {
        return showMainMenu();
    }

    const { label, text, emoji } = await inquirer.prompt([
        {
            type: 'input',
            name: 'label',
            message: 'Template Name:',
            validate: (input: string) => input.length > 0 || 'Name ist erforderlich'
        },
        {
            type: 'input',
            name: 'text',
            message: 'Status Text:',
            validate: (input: string) => input.length > 0 || 'Text ist erforderlich'
        },
        {
            type: 'input',
            name: 'emoji',
            message: 'Emoji:',
            validate: (input: string) => input.length > 0 || 'Emoji ist erforderlich'
        }
    ]);

    const templates = loadTemplates();
    templates.push({ label, text, emoji });
    saveTemplates(templates);
    console.log(chalk.green('✓ Template erfolgreich erstellt!'));
    showMainMenu();
}

async function deleteTemplate() {
    await clearConsole();
    const templates = loadTemplates();
    
    const { template } = await inquirer.prompt([
        {
            type: 'list',
            name: 'template',
            message: 'Welches Template möchtest du löschen?',
            choices: [
                ...templates.map(t => ({
                    name: t.label,
                    value: t
                })),
                new inquirer.Separator(),
                { name: '↩️  Zurück zum Hauptmenü', value: 'back' }
            ]
        }
    ]);

    if (template === 'back') {
        return showMainMenu();
    }

    const updatedTemplates = templates.filter(t => t.label !== template.label);
    saveTemplates(updatedTemplates);
    console.log(chalk.green('✓ Template erfolgreich gelöscht!'));
    showMainMenu();
}

console.log(chalk.blue.bold('🎯 Slack Status Manager wird gestartet...'));
startApplication().catch(error => {
    console.error(chalk.red('Fehler beim Starten der Anwendung:', error));
    process.exit(1);
});