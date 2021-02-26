const ErrorHandling = artifacts.require("ErrorHandling");
const Panic = artifacts.require("Panic");
const {
    expectRevert, // Assertions for transactions that should fail
} = require('@openzeppelin/test-helpers');

contract('ErrorHandling', function (accounts) {
    let errorHandling;
    before(async function () {
        let panic = await Panic.new();
        errorHandling = await ErrorHandling.new(panic.address);
    })

    it("sub-call bubbles up normally", async function () {
        await expectRevert(errorHandling.execute(), "panic");
    })

    it("low-level call won't bubble up", async function () {
        await errorHandling.lowLevelCall();
    })

    it("try/catch won't bubble up", async function () {
        await errorHandling.tryCatch();
    })
})