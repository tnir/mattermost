// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import {FormattedMessage, useIntl} from 'react-intl';

import type {PreferenceType} from '@mattermost/types/preferences';

import {Preferences} from 'utils/constants';

import SettingItemMax from 'components/setting_item_max';
import SettingItemMin from 'components/setting_item_min';

import type {ActionResult} from 'mattermost-redux/types/actions';

export type Props = {
    currentUserId: string;
    preferences: {[key: string]: PreferenceType};
    updateSection: (section: string) => void;
    active: boolean;
    areAllSectionsInactive: boolean;
    actions: {
        savePreferences: (userId: string, preferences: PreferenceType[]) => Promise<ActionResult>;
    };
};

const ManageLinkPreviewDomains: React.FC<Props> = ({
    currentUserId,
    preferences,
    updateSection,
    active,
    areAllSectionsInactive,
    actions,
}) => {
    const intl = useIntl();
    const [domainInput, setDomainInput] = useState('');
    const [saving, setSaving] = useState(false);
    const [serverError, setServerError] = useState('');

    // Get current domain preferences
    const domainPreferences: {[domain: string]: boolean} = {};
    Object.values(preferences).forEach((pref) => {
        if (pref.category === Preferences.CATEGORY_LINK_PREVIEW_DOMAIN_SETTINGS) {
            domainPreferences[pref.name] = pref.value === 'true';
        }
    });

    const handleAddDomain = async () => {
        if (!domainInput.trim()) {
            return;
        }

        const domain = domainInput.trim().toLowerCase();
        
        // Basic domain validation
        if (!/^[a-zA-Z0-9][a-zA-Z0-9-._]+[a-zA-Z0-9]$/.test(domain)) {
            setServerError(intl.formatMessage({
                id: 'user.settings.display.linkPreviewDomains.invalidDomain',
                defaultMessage: 'Please enter a valid domain name.',
            }));
            return;
        }

        setSaving(true);
        setServerError('');

        const preference: PreferenceType = {
            user_id: currentUserId,
            category: Preferences.CATEGORY_LINK_PREVIEW_DOMAIN_SETTINGS,
            name: domain,
            value: 'false', // Default to disabled
        };

        const result = await actions.savePreferences(currentUserId, [preference]);
        if (result.error) {
            setServerError(result.error.message || 'Failed to save preference');
        } else {
            setDomainInput('');
        }
        setSaving(false);
    };

    const handleToggleDomain = async (domain: string, enabled: boolean) => {
        setSaving(true);
        setServerError('');

        const preference: PreferenceType = {
            user_id: currentUserId,
            category: Preferences.CATEGORY_LINK_PREVIEW_DOMAIN_SETTINGS,
            name: domain,
            value: enabled ? 'true' : 'false',
        };

        const result = await actions.savePreferences(currentUserId, [preference]);
        if (result.error) {
            setServerError(result.error.message || 'Failed to save preference');
        }
        setSaving(false);
    };

    const handleRemoveDomain = async (domain: string) => {
        setSaving(true);
        setServerError('');

        const preference: PreferenceType = {
            user_id: currentUserId,
            category: Preferences.CATEGORY_LINK_PREVIEW_DOMAIN_SETTINGS,
            name: domain,
            value: '',
        };

        const result = await actions.savePreferences(currentUserId, [preference]);
        if (result.error) {
            setServerError(result.error.message || 'Failed to remove preference');
        }
        setSaving(false);
    };

    const handleSubmit = () => {
        updateSection('');
    };

    if (active) {
        const inputs = [
            <div key='domain-input'>
                <label className='control-label'>
                    <FormattedMessage
                        id='user.settings.display.linkPreviewDomains.addDomain'
                        defaultMessage='Add Domain'
                    />
                </label>
                <div className='input-group'>
                    <input
                        type='text'
                        className='form-control'
                        value={domainInput}
                        onChange={(e) => setDomainInput(e.target.value)}
                        placeholder={intl.formatMessage({
                            id: 'user.settings.display.linkPreviewDomains.placeholder',
                            defaultMessage: 'e.g., example.com',
                        })}
                        onKeyPress={(e) => {
                            if (e.key === 'Enter') {
                                handleAddDomain();
                            }
                        }}
                    />
                    <span className='input-group-btn'>
                        <button
                            type='button'
                            className='btn btn-primary'
                            onClick={handleAddDomain}
                            disabled={saving || !domainInput.trim()}
                        >
                            <FormattedMessage
                                id='user.settings.display.linkPreviewDomains.add'
                                defaultMessage='Add'
                            />
                        </button>
                    </span>
                </div>
            </div>,
        ];

        // Add current domains list
        const domainsList = Object.keys(domainPreferences).map((domain) => (
            <div
                key={domain}
                className='form-group'
            >
                <div className='checkbox'>
                    <label>
                        <input
                            type='checkbox'
                            checked={domainPreferences[domain]}
                            onChange={(e) => handleToggleDomain(domain, e.target.checked)}
                            disabled={saving}
                        />
                        <span className='domain-name'>{domain}</span>
                    </label>
                    <button
                        type='button'
                        className='btn btn-sm btn-link text-danger pull-right'
                        onClick={() => handleRemoveDomain(domain)}
                        disabled={saving}
                    >
                        <FormattedMessage
                            id='user.settings.display.linkPreviewDomains.remove'
                            defaultMessage='Remove'
                        />
                    </button>
                </div>
            </div>
        ));

        if (domainsList.length > 0) {
            inputs.push(
                <div key='domains-list'>
                    <label className='control-label'>
                        <FormattedMessage
                            id='user.settings.display.linkPreviewDomains.manageDomains'
                            defaultMessage='Manage Domains (uncheck to disable previews)'
                        />
                    </label>
                    <div className='domains-list'>
                        {domainsList}
                    </div>
                </div>,
            );
        }

        const extraInfo = (
            <span>
                <FormattedMessage
                    id='user.settings.display.linkPreviewDomains.help'
                    defaultMessage='Control which domains show link previews. Unchecked domains will not show previews, even if link previews are enabled.'
                />
            </span>
        );

        return (
            <div>
                <SettingItemMax
                    title={intl.formatMessage({
                        id: 'user.settings.display.linkPreviewDomains.title',
                        defaultMessage: 'Link Preview Domains',
                    })}
                    inputs={inputs}
                    submit={handleSubmit}
                    saving={saving}
                    serverError={serverError}
                    extraInfo={extraInfo}
                    updateSection={updateSection}
                />
                <div className='divider-dark'/>
            </div>
        );
    }

    const numDomains = Object.keys(domainPreferences).length;
    const describe = numDomains > 0 ? 
        intl.formatMessage({
            id: 'user.settings.display.linkPreviewDomains.describe',
            defaultMessage: '{count, plural, one {# domain} other {# domains}} configured',
        }, {count: numDomains}) :
        intl.formatMessage({
            id: 'user.settings.display.linkPreviewDomains.describeEmpty',
            defaultMessage: 'No domains configured',
        });

    return (
        <div>
            <SettingItemMin
                title={intl.formatMessage({
                    id: 'user.settings.display.linkPreviewDomains.title',
                    defaultMessage: 'Link Preview Domains',
                })}
                describe={describe}
                section='linkPreviewDomains'
                updateSection={updateSection}
            />
            <div className='divider-dark'/>
        </div>
    );
};

export default ManageLinkPreviewDomains;