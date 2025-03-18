#include <systemc.h>

const auto delay = sc_time(3.0, SC_NS);

// DM(name, msg) produces the two string arguments to stdout without an endl.
// DME(name, msg) is like DM(), but with an endl.
// DMS(scm, msg) prints scm->name() as the name without an endl.
// DMSE(scm, msg) is like DMS(), but with an endl.
// DBS(msg) is like DMS() with sc_module implicitly "this", no endl.
// DBSE(msg) is like DMSE(), but the sc_module is implicitly "this".

#define DM(name, msg) cout << (name) << "@" << sc_time_stamp() << ": " << msg
#define DME(name, msg) DM(name, msg) << endl
#define DMS(scm, msg) DM(scm->name(), msg)
#define DMSE(scm, msg) DME(scm->name(), msg)
#define DBS(msg) DMS(this, msg)
#define DBSE(msg) DMSE(this, msg)

// Asynchronous adder with output propagation delay
SC_MODULE(adder4) {
    sc_in<sc_uint<4>>   A,B;
    sc_out<sc_uint<4>>  OUT;

    void add() {
        while(true) {
            DBSE("waiting for input...");
            wait(A.default_event() | B.default_event());
            DBS("summing (") << A.read() << "+" << B.read() << ")" << endl;
            auto sum = A.read() + B.read();  // calculate sum
            DBSE("delaying...");
            wait(delay);                     // wait delay
            DBS("producing: ") << sum << endl;
            OUT.write(sum);                  // write sum after delay
        }
    }

    SC_CTOR(adder4) {
        SC_THREAD(add);  //thread instead of process         
    }
};

SC_MODULE(addBetter) {
    sc_in<sc_uint<4>>   A,B;
    sc_out<sc_uint<4>>  OUT;

    void inputsChanged() {
    }

    void produce() {
    }

    SC_CTOR(addBetter) {
        SC_METHOD(add);
    }
}

SC_MODULE(driver) {
    unsigned a,b;                //internal data values
    sc_in<bool> clk_in;          //system clock  input
    sc_out<sc_uint<4>> out_a, out_b; //driver data outputs

    void proc() {
        out_a.write(a);
        wait(1, SC_NS);
        out_b.write(b);

        //change internal data to drive test device. just increment
        a += 1;
        b += 2;
    }
    SC_CTOR(driver) {
        a = b = 0;
        SC_METHOD(proc);
        sensitive << clk_in.pos();
        dont_initialize();
    }
};

int sc_main(int, char**) {
    sc_clock clk("clk", 5, SC_NS); //system clock
    driver drv("driver");          //test case driver
    adder4 adder("adder");         //device under test

    //connect devices
    sc_signal<sc_uint<4>> s1, s2, s3;
    drv.clk_in(clk);
    drv.out_a(s1);
    drv.out_b(s2);
    adder.A(drv.out_a);
    adder.B(drv.out_b);
    adder.OUT(s3);
    
    sc_start(40, SC_NS);
    return 0;
}
