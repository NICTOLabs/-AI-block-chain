// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Pausable.sol";
import "@openzeppelin/contracts/access/AccessControl.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

contract WTDR is ERC20, ERC20Burnable, ERC20Pausable, AccessControl, ReentrancyGuard {
    bytes32 public constant BRIDGE_ROLE = keccak256("BRIDGE_ROLE");
    bytes32 public constant PAUSER_ROLE = keccak256("PAUSER_ROLE");
    bytes32 public constant MINTER_ROLE = keccak256("MINTER_ROLE");

    event Mint(address indexed to, uint256 amount);
    event Burn(address indexed from, uint256 amount);
    event BridgeLock(address indexed from, uint256 amount, bytes32 indexed sourceTxHash);
    event BridgeMint(address indexed to, uint256 amount, bytes32 indexed sourceTxHash);
    event BridgeRelease(address indexed to, uint256 amount, bytes32 indexed sourceTxHash);
    event Paused(address account);
    event Unpaused(address account);

    mapping(address => uint256) public nodeBalances;
    mapping(bytes32 => bool) public processedBridges;
    uint256 public constant MAX_SUPPLY = 10_000_000_000 * 10**8;
    uint256 public totalMinted;

    constructor(
        string memory name,
        string memory symbol,
        address bridgeOperator,
        address admin
    ) ERC20(name, symbol) {
        if (bridgeOperator == address(0) || admin == address(0)) {
            revert ZeroAddress();
        }

        _grantRole(DEFAULT_ADMIN_ROLE, admin);
        _grantRole(BRIDGE_ROLE, bridgeOperator);
        _grantRole(PAUSER_ROLE, admin);
        _grantRole(MINTER_ROLE, admin);

        _transferOwnership(admin);
    }

    modifier onlyBridge() {
        require(hasRole(BRIDGE_ROLE, msg.sender), "WTDR: caller is not bridge");
        _;
    }

    modifier onlyPauser() {
        require(hasRole(PAUSER_ROLE, msg.sender), "WTDR: caller is not pauser");
        _;
    }

    function mint(address to, uint256 amount) external onlyMinter nonReentrant returns (bool) {
        require(to != address(0), "WTDR: mint to zero address");
        require(totalSupply() + amount <= MAX_SUPPLY, "WTDR: exceeds max supply");

        totalMinted += amount;
        _mint(to, amount);

        emit Mint(to, amount);
        return true;
    }

    function mintToNode(address node, uint256 amount) external onlyMinter nonReentrant returns (bool) {
        require(node != address(0), "WTDR: mint to zero address");
        require(totalSupply() + amount <= MAX_SUPPLY, "WTDR: exceeds max supply");

        totalMinted += amount;
        nodeBalances[node] += amount;
        _mint(node, amount);

        emit Mint(node, amount);
        return true;
    }

    function burnFromNode(address node, uint256 amount) external onlyMinter nonReentrant returns (bool) {
        require(node != address(0), "WTDR: burn from zero address");
        require(nodeBalances[node] >= amount, "WTDR: insufficient node balance for burn");

        nodeBalances[node] -= amount;
        totalMinted -= amount;
        _burn(node, amount);

        emit Burn(node, amount);
        return true;
    }

    function bridgeLock(bytes32 sourceTxHash, bytes calldata sourceTxProof) external onlyBridge nonReentrant returns (bool) {
        require(!processedBridges[sourceTxHash], "WTDR: bridge tx already processed");

        address from = abi.decode(sourceTxProof, (address));
        uint256 amount = abi.decode(sourceTxProof[32:], (uint256));

        processedBridges[sourceTxHash] = true;

        uint256 lockedBalance = nodeBalances[from] + amount;
        nodeBalances[from] = lockedBalance;

        emit BridgeLock(from, amount, sourceTxHash);
        return true;
    }

    function bridgeMint(address to, uint256 amount, bytes32 sourceTxHash) external onlyBridge nonReentrant returns (bool) {
        require(!processedBridges[sourceTxHash], "WTDR: bridge tx already processed");
        require(to != address(0), "WTDR: mint to zero address");
        require(totalSupply() + amount <= MAX_SUPPLY, "WTDR: exceeds max supply");

        processedBridges[sourceTxHash] = true;
        totalMinted += amount;
        _mint(to, amount);

        emit BridgeMint(to, amount, sourceTxHash);
        return true;
    }

    function bridgeRelease(address to, uint256 amount, bytes32 sourceTxHash) external onlyBridge nonReentrant returns (bool) {
        require(!processedBridges[sourceTxHash], "WTDR: bridge tx already processed");
        require(to != address(0), "WTDR: release to zero address");
        require(totalSupply() >= amount, "WTDR: insufficient total supply for release");

        processedBridges[sourceTxHash] = true;
        totalMinted -= amount;

        uint256 burnAmount = (amount * 1) / 1000;
        uint256 releaseAmount = amount - burnAmount;

        _burn(to, burnAmount);
        _transfer(address(this), to, releaseAmount);

        emit BridgeRelease(to, amount, sourceTxHash);
        return true;
    }

    function pause() external onlyPauser {
        _pause();
    }

    function unpause() external onlyPauser {
        _unpause();
    }

    function grantRole(bytes32 role, address account) external onlyRole(DEFAULT_ADMIN_ROLE) {
        _grantRole(role, account);
    }

    function revokeRole(bytes32 role, address account) external onlyRole(DEFAULT_ADMIN_ROLE) {
        _revokeRole(role, account);
    }

    function getNodeBalance(address node) external view returns (uint256) {
        return nodeBalances[node];
    }

    function totalSupply() public view override(ERC20, IERC20) returns (uint256) {
        return super.totalSupply();
    }

    function _update(address from, address to, uint256 value) internal override(ERC20, ERC20Pausable) {
        if (paused()) {
            require(from == address(0) || to == address(0), "WTDR: token transfers are paused");
        }
        super._update(from, to, value);
    }

    function supportsInterface(bytes4 interfaceId) public view override(AccessControl) returns (bool) {
        return super.supportsInterface(interfaceId);
    }
}
