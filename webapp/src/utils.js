export function getPluginURL() {
    const siteURL = window.basename || '';
    return `${siteURL}/plugins/com.mattermost.clickup`;
}
