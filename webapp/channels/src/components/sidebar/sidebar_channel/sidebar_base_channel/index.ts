// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import type {ConnectedProps} from 'react-redux';
import {bindActionCreators} from 'redux';
import type {Dispatch} from 'redux';

import type {Channel} from '@mattermost/types/channels';

import {leaveChannel} from 'actions/views/channel';
import {openModal} from 'actions/views/modals';

import {isOfficialTunagChannel} from 'utils/official_channel_utils';

import type {GlobalState} from 'types/store';

import SidebarBaseChannel from './sidebar_base_channel';

export type OwnProps = {
    channel: Channel;
    currentTeamName: string;
}

function mapStateToProps(state: GlobalState, ownProps: OwnProps) {
    return {
        isOfficial: isOfficialTunagChannel(ownProps.channel, state),
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        actions: bindActionCreators({
            leaveChannel,
            openModal,
        }, dispatch),
    };
}

const connector = connect(mapStateToProps, mapDispatchToProps);

export type PropsFromRedux = ConnectedProps<typeof connector>;

export default connector(SidebarBaseChannel);
