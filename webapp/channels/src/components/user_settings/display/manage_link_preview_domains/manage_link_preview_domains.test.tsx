// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Provider} from 'react-redux';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import {shallow} from 'enzyme';

import {TestHelper} from 'utils/test_helper';

import ManageLinkPreviewDomains from './manage_link_preview_domains';

describe('components/user_settings/display/manage_link_preview_domains/manage_link_preview_domains', () => {
    const mockStore = configureStore([thunk]);

    const defaultProps = {
        currentUserId: 'user1',
        preferences: {},
        updateSection: jest.fn(),
        active: false,
        areAllSectionsInactive: false,
        actions: {
            savePreferences: jest.fn().mockResolvedValue({data: true}),
        },
    };

    test('should render when inactive', () => {
        const wrapper = shallow(
            <ManageLinkPreviewDomains {...defaultProps}/>,
        );

        expect(wrapper).toMatchSnapshot();
    });

    test('should render when active', () => {
        const props = {
            ...defaultProps,
            active: true,
        };

        const wrapper = shallow(
            <ManageLinkPreviewDomains {...props}/>,
        );

        expect(wrapper).toMatchSnapshot();
    });

    test('should display existing domain preferences', () => {
        const preferences = {
            'link_preview_domain_settings--example.com': {
                user_id: 'user1',
                category: 'link_preview_domain_settings',
                name: 'example.com',
                value: 'false',
            },
            'link_preview_domain_settings--test.com': {
                user_id: 'user1',
                category: 'link_preview_domain_settings',
                name: 'test.com',
                value: 'true',
            },
        };

        const props = {
            ...defaultProps,
            preferences,
            active: true,
        };

        const wrapper = shallow(
            <ManageLinkPreviewDomains {...props}/>,
        );

        // Should show 2 domains configured
        expect(wrapper.find('.domains-list')).toHaveLength(1);
    });
});