// SPDX-License-Identifier: MIT

pragma solidity ^0.7.0;

interface IPanic {
    function panic() external;
}

contract ErrorHandling {
    IPanic private p;

    constructor(IPanic _p) {
        p = _p;
    }

    /// @dev this should panic
    function execute() external {
        p.panic();
    }

    /// @dev this won't panic
    function lowLevelCall() external {
        address(p).call(abi.encodeWithSignature("panic()"));
    }

    /// @dev this won't panic
    function tryCatch() external {
        try p.panic() {} catch {}
    }
}

contract Panic {
    uint private a;

    function panic() external {
        a = 0;
        require(false, 'panic');
    }
}
