// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract TDRWrapped is ERC20, Ownable {
    constructor() ERC20("TDR Wrapped", "wTDR") Ownable(msg.sender) {
        _mint(msg.sender, 1_000_000_000 * 10**decimals());
    }

    function mint(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
        emit Mint(to, amount);
    }

    function burn(address from, uint256 amount) external onlyOwner {
        _burn(from, amount);
        emit Burn(from, amount);
    }

    event Mint(address indexed to, uint256 amount);
    event Burn(address indexed from, uint256 amount);
}
