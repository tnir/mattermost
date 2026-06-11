// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {act, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import type {ChannelType} from '@mattermost/types/channels';

import {renderWithContext} from 'tests/react_testing_utils';
import * as officialChannelUtils from 'utils/official_channel_utils';
import {TestHelper} from 'utils/test_helper';

import ChannelSettingsInfoTab from './channel_settings_info_tab';

// Mock the redux actions and selectors
jest.mock('mattermost-redux/actions/channels', () => ({
    patchChannel: jest.fn(),
    updateChannelPrivacy: jest.fn(),
}));

// Mock the ConvertConfirmModal component
jest.mock('components/admin_console/team_channel_settings/convert_confirm_modal', () => {
    return jest.fn().mockImplementation(({show, onCancel, onConfirm, displayName}) => {
        if (!show) {
            return null;
        }
        return (
            <div data-testid='convert-confirm-modal'>
                <div>{'Converting '}{displayName}</div>
                <button onClick={onCancel}>{'Cancel'}</button>
                <button onClick={onConfirm}>{'Yes, Convert Channel'}</button>
            </div>
        );
    });
});

let mockChannelPropertiesPermission = true;
let mockConvertToPublicPermission = true;
let mockConvertToPrivatePermission = true;

jest.mock('mattermost-redux/selectors/entities/roles', () => ({
    haveITeamPermission: jest.fn().mockReturnValue(true),
    haveIChannelPermission: jest.fn().mockImplementation((state, teamId, channelId, permission: string) => {
        if (permission === 'manage_private_channel_properties' || permission === 'manage_public_channel_properties') {
            return mockChannelPropertiesPermission;
        }
        if (permission === 'convert_public_channel_to_private') {
            return mockConvertToPrivatePermission;
        }
        if (permission === 'convert_private_channel_to_public') {
            return mockConvertToPublicPermission;
        }
        return true;
    }),
    getRoles: jest.fn().mockReturnValue({}),
}));

jest.mock('selectors/views/textbox', () => ({
    showPreviewOnChannelSettingsHeaderModal: jest.fn().mockReturnValue(false),
    showPreviewOnChannelSettingsPurposeModal: jest.fn().mockReturnValue(false),
}));

jest.mock('actions/views/textbox', () => ({
    setShowPreviewOnChannelSettingsHeaderModal: jest.fn(),
    setShowPreviewOnChannelSettingsPurposeModal: jest.fn(),
}));

// Mock the current user
const mockUser = TestHelper.getUserMock({
    id: 'user_id',
    roles: 'system_admin',
});

jest.mock('mattermost-redux/selectors/entities/channels', () => ({
    ...jest.requireActual('mattermost-redux/selectors/entities/channels') as typeof import('mattermost-redux/selectors/entities/channels'),
    getChannelMember: jest.fn(() => ({roles: 'channel_user system_admin'})),
}));

jest.mock('mattermost-redux/selectors/entities/common', () => {
    return {
        ...jest.requireActual('mattermost-redux/selectors/entities/common') as typeof import('mattermost-redux/selectors/entities/users'),
        getCurrentUser: () => mockUser,
    };
});

// Mock channel for testing (Private by default for these tests)
const mockPrivateChannel = TestHelper.getChannelMock({
    id: 'channel1',
    team_id: 'team1',
    display_name: 'Test Private Channel',
    name: 'test-private-channel',
    type: 'P' as ChannelType,
});

const baseProps = {
    channel: mockPrivateChannel,
    setAreThereUnsavedChanges: jest.fn(),
};

describe('ChannelSettingsInfoTab Private to Public Conversion (Engage Chat)', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        mockChannelPropertiesPermission = true;
        mockConvertToPublicPermission = true;
        mockConvertToPrivatePermission = true;
    });

    it('should show ConvertConfirmModal when converting from private to public', async () => {
        mockConvertToPublicPermission = true;

        renderWithContext(<ChannelSettingsInfoTab {...baseProps}/>);

        // Change to public channel
        const publicButton = screen.getByRole('button', {name: /Public Channel/});
        await userEvent.click(publicButton);

        // Click Save button
        await act(async () => {
            await userEvent.click(screen.getByRole('button', {name: 'Save'}));
        });

        // Verify the modal is shown
        expect(screen.getByTestId('convert-confirm-modal')).toBeInTheDocument();
    });

    it('should convert channel when confirming in ConvertConfirmModal (private to public)', async () => {
        mockConvertToPublicPermission = true;

        const {updateChannelPrivacy, patchChannel} = require('mattermost-redux/actions/channels');
        updateChannelPrivacy.mockReturnValue({type: 'MOCK_ACTION', data: {}});
        patchChannel.mockReturnValue({type: 'MOCK_ACTION', data: {}});

        renderWithContext(<ChannelSettingsInfoTab {...baseProps}/>);

        // Change to public channel
        const publicButton = screen.getByRole('button', {name: /Public Channel/});
        await userEvent.click(publicButton);

        // Click Save button to show modal
        await act(async () => {
            await userEvent.click(screen.getByRole('button', {name: 'Save'}));
        });

        // Click confirm button in modal
        await act(async () => {
            await userEvent.click(screen.getByText(/Yes, Convert Channel/i));
        });

        // Verify updateChannelPrivacy was called with 'O' (Open)
        expect(updateChannelPrivacy).toHaveBeenCalledWith('channel1', 'O');
    });

    it('should not convert channel when canceling in ConvertConfirmModal (private to public)', async () => {
        mockConvertToPublicPermission = true;

        const {updateChannelPrivacy} = require('mattermost-redux/actions/channels');
        updateChannelPrivacy.mockReturnValue({type: 'MOCK_ACTION', data: {}});

        renderWithContext(<ChannelSettingsInfoTab {...baseProps}/>);

        // Change to public channel
        const publicButton = screen.getByRole('button', {name: /Public Channel/});
        await userEvent.click(publicButton);

        // Click Save button to show modal
        await act(async () => {
            await userEvent.click(screen.getByRole('button', {name: 'Save'}));
        });

        // Click cancel button in modal
        await act(async () => {
            await userEvent.click(screen.getByText(/Cancel/i));
        });

        // Verify updateChannelPrivacy was not called
        expect(updateChannelPrivacy).not.toHaveBeenCalled();
    });

    it('should handle errors when converting channel privacy (private to public)', async () => {
        mockConvertToPublicPermission = true;

        const {updateChannelPrivacy} = require('mattermost-redux/actions/channels');
        updateChannelPrivacy.mockReturnValue({
            type: 'MOCK_ACTION',
            error: {message: 'Error changing privacy'},
        });

        renderWithContext(<ChannelSettingsInfoTab {...baseProps}/>);

        // Change to public channel
        const publicButton = screen.getByRole('button', {name: /Public Channel/});
        await userEvent.click(publicButton);

        // Click Save button to show modal
        await act(async () => {
            await userEvent.click(screen.getByRole('button', {name: 'Save'}));
        });

        // Click confirm button in modal
        await act(async () => {
            await userEvent.click(screen.getByText(/Yes, Convert Channel/i));
        });

        // Verify error state is shown in the save panel
        expect(screen.getByText(/There are errors in the form above/)).toBeInTheDocument();
    });

    it('should disable privacy conversion buttons when the channel is official', () => {
        const spy = jest.spyOn(officialChannelUtils, 'isOfficialTunagChannel').mockReturnValue(true);

        renderWithContext(<ChannelSettingsInfoTab {...baseProps}/>);

        // Both public and private buttons should be disabled for official channels
        expect(screen.getByRole('button', {name: /Public Channel/})).toHaveClass('disabled');
        expect(screen.getByRole('button', {name: /Private Channel/})).toHaveClass('disabled');

        spy.mockRestore();
    });
});
