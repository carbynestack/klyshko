/*
 * Copyright (c) 2022 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
module.exports = {
    extends: [
        "@commitlint/config-conventional"
    ],
    rules: {
        'scope-empty': [0, 'never'],
        "scope-enum": [
            2,
            "always",
            [
                "mp-spdz",
                "operator",
                "provisioner"
            ]
        ]
    }
}
