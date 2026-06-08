// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getTranslations, customTranslations} from 'selectors/i18n';

describe('selectors/i18n (engage-chat overrides)', () => {
    let originalEnCustom;
    let originalJaCustom;

    beforeAll(() => {
        originalEnCustom = customTranslations.en;
        originalJaCustom = customTranslations.ja;

        customTranslations.en = {};
        customTranslations.ja = {
            'test.custom_key': 'オーバーライド',
        };
    });

    afterAll(() => {
        customTranslations.en = originalEnCustom;
        customTranslations.ja = originalJaCustom;
    });

    test('merges custom translations correctly', () => {
        const jaState = {
            views: {
                i18n: {
                    translations: {
                        ja: {
                            'test.hello_world': 'こんにちは世界',
                        },
                    },
                },
            },
        };

        const result = getTranslations(jaState, 'ja');
        expect(result['test.hello_world']).toEqual('こんにちは世界');
        expect(result['test.custom_key']).toEqual('オーバーライド');
    });

    test('returns the cached translations object on consecutive calls to preserve reference equality', () => {
        const jaState = {
            views: {
                i18n: {
                    translations: {
                        ja: {
                            'test.hello_world': 'こんにちは世界',
                        },
                    },
                },
            },
        };

        const firstCall = getTranslations(jaState, 'ja');
        const secondCall = getTranslations(jaState, 'ja');

        expect(firstCall).toBe(secondCall);
    });
});
