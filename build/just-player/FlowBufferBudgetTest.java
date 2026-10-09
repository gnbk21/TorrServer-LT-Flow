package com.brouken.player;

public final class FlowBufferBudgetTest {
    public static void main(String[] args) {
        long m = 1024L*1024;
        if (FlowBufferBudget.bytes(1024*m,64*m,8192*m,false) <= 144*m) throw new AssertionError("healthy device receives no extra reserve");
        if (FlowBufferBudget.bytes(8192*m,0,32768*m,false) > 384*m) throw new AssertionError("unbounded target");
        if (FlowBufferBudget.bytes(1024*m,64*m,8192*m,true) > 32*m) throw new AssertionError("low RAM ignored");
        if (FlowBufferBudget.bytes(512*m,450*m,128*m,false) > 8*m) throw new AssertionError("heap pressure ignored");
        if (FlowBufferBudget.bytes(0,0,0,false) != m) throw new AssertionError("minimum allocator invalid");
        System.out.println("Device and system memory budgets passed");
    }
}
