import {FormattedMessage} from 'react-intl';
import {Client4} from 'mattermost-redux/client';
import {getPluginURL} from './utils';
import TasksRHS from './components/tasks_rhs';

const React = window.React;

export default class ClickUpPlugin {
    initialize(registry, store) {
        const pluginURL = getPluginURL();

        registry.registerPostDropdownMenuAction(
            <FormattedMessage
                id='plugin.post_action'
                defaultMessage='Create ClickUp Task'
            />,
            (postId) => {
                const post = store.getState().entities.posts.posts[postId];
                if (!post) {
                    return;
                }

                const title = (post.message || '').split('\n')[0].trim().slice(0, 100);
                if (!title) {
                    return;
                }

                Client4.executeCommand(post.channel_id, `/clickup task ${title}`);
            },
        );

        const rhs = registry.registerRightHandSidebarComponent({
            title: (
                <FormattedMessage
                    id='plugin.rhs.title'
                    defaultMessage='ClickUp Tasks'
                />
            ),
            component: TasksRHS,
        });

        registry.registerAppBarComponent({
            iconUrl: `${pluginURL}/public/clickup-icon.svg`,
            action: () => {
                const channelId = store.getState().entities.channels.currentChannelId;
                if (channelId) {
                    store.dispatch(rhs.toggleRHSPlugin);
                }
            },
            tooltipText: (
                <FormattedMessage
                    id='plugin.app_bar'
                    defaultMessage='ClickUp Tasks'
                />
            ),
        });

        registry.registerChannelHeaderButtonAction(
            <i className='icon fa fa-check-square-o'/>,
            () => store.dispatch(rhs.toggleRHSPlugin),
            'ClickUp Tasks',
        );
    }
}
