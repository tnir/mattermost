// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

type Props = {
    className?: string;
    size?: number;
}

// Font Awesome Free v7.2.0 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2026 Fonticons, Inc.
const BUILDING_ICON_PATH = 'M64 48c-8.8 0-16 7.2-16 16l0 384c0 8.8 7.2 16 16 16l80 0 0-64c0-26.5 21.5-48 48-48s48 21.5 48 48l0 64 80 0c8.8 0 16-7.2 16-16l0-384c0-8.8-7.2-16-16-16L64 48zM0 64C0 28.7 28.7 0 64 0L320 0c35.3 0 64 28.7 64 64l0 384c0 35.3-28.7 64-64 64L64 512c-35.3 0-64-28.7-64-64L0 64zm88 40c0-8.8 7.2-16 16-16l48 0c8.8 0 16 7.2 16 16l0 48c0 8.8-7.2 16-16 16l-48 0c-8.8 0-16-7.2-16-16l0-48zM232 88l48 0c8.8 0 16 7.2 16 16l0 48c0 8.8-7.2 16-16 16l-48 0c-8.8 0-16-7.2-16-16l0-48c0-8.8 7.2-16 16-16zM88 232c0-8.8 7.2-16 16-16l48 0c8.8 0 16 7.2 16 16l0 48c0 8.8-7.2 16-16 16l-48 0c-8.8 0-16-7.2-16-16l0-48zm144-16l48 0c8.8 0 16 7.2 16 16l0 48c0 8.8-7.2 16-16 16l-48 0c-8.8 0-16-7.2-16-16l0-48c0-8.8 7.2-16 16-16z';

export default function BuildingIcon({className, size}: Props) {
    if (size) {
        return (
            <svg
                viewBox='0 0 384 512'
                role='img'
                aria-label='Building icon'
                fill='currentColor'
                className={className}
                style={{
                    width: size,
                    height: size,
                }}
            >
                <path d={BUILDING_ICON_PATH}/>
            </svg>
        );
    }

    return (
        <i
            className={className || 'icon'}
            style={{
                display: 'flex',
                width: '1em',
                height: '1em',
                minWidth: '1em',
                flexShrink: 0,
                alignItems: 'center',
                justifyContent: 'center',
                marginLeft: 'calc(-2px + 0.2em)',
                marginRight: 'calc(6px + 0.2em)',
            }}
        >
            <svg
                viewBox='0 0 384 512'
                role='img'
                aria-label='Building icon'
                fill='currentColor'
                style={{
                    width: '0.9em',
                    height: '0.9em',
                }}
            >
                <path d={BUILDING_ICON_PATH}/>
            </svg>
        </i>
    );
}
