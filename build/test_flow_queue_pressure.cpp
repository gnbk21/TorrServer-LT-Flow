// Compile against the actual reviewed native header, not a policy reimplementation.
#include "libtorrent/aux_/flow_stream_policy.hpp"
#include <cassert>
#include <iostream>

using libtorrent::aux::flow_service_model;

int main()
{
    flow_service_model congested;
    for (int i=0;i<32;++i) congested.observe(30000,1000+i*100);
    int credit=512;
    for (int i=0;i<12;++i) {
        auto const now=4200+i*500;
        congested.observe(30000,now);
        int const next=congested.credit(credit,65536,16384,512,1500,now,credit);
        assert(next<=credit);
        credit=next;
    }
    assert(credit<=32);
    assert(!congested.urgent_work_allowed(70000,16,512,true,9700));

    // A genuinely delayed, underfilled path still escapes the two-block window.
    flow_service_model delayed;
    for (int i=0;i<32;++i) delayed.observe(2500,1000+i*100);
    assert(delayed.credit(2,12000,16384,512,512,4200,2)>2);
    assert(delayed.urgent_work_allowed(4000,2,4,true,4200));
    // A full-window allowance must not admit tens of seconds of queued work
    // just because the peer's desired slot count remains high after a rate drop.
    assert(!delayed.urgent_work_allowed(25000,100,428,true,4200));
    auto const horizon=delayed.urgent_horizon_ms(true,4200);
    assert(delayed.urgent_work_allowed(2*horizon,2,4,true,4200));
    assert(!delayed.urgent_work_allowed(2*horizon+1,2,4,true,4200));
    // Waiting for a congested peer must not prevent use of another free peer.
    flow_service_model fast;
    for (int i=0;i<32;++i) fast.observe(100,1000+i*100);
    assert(fast.urgent_work_allowed(500,1,8,true,4200));
    assert(delayed.credit(428,12000,16384,512,512,4800,0)==428);
    assert(delayed.credit(428,12000,16384,32,512,5400,0)==32);
    assert(!delayed.urgent_work_allowed(70000,2,4,false,5400));
    assert(!delayed.urgent_work_allowed(70000,2,4,true,30000));
    assert(delayed.request_timeout_seconds(2,60,true,5400)==4);
    std::cout << "queue pressure, delayed bootstrap, idle transition and limits passed\n";
}
