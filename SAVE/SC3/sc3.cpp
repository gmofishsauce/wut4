#include <systemc.h>

using data_t = sc_uint<4>;
const auto delay = sc_time(2.0, SC_NS);

void dbs(const char* name, const char* msg) {
    cout << name << "@" << sc_time_stamp() << ": " << msg << endl;
}

void db(sc_module* scm, const char* msg) {
    dbs(scm->name(), msg);
}

#define DB(msg) db(this, msg)
#define DBS(name, msg) dbs(name, msg)

// Asynchronous adder with output propagation delay
SC_MODULE(adder4) {
    sc_in<data_t>   A,B;
    sc_out<data_t>  OUT;

    void add() {
        while(true) {
            DB("waiting for input...");
            wait(A.default_event() | B.default_event());
            DB("summing...");
            auto sum = A.read() + B.read();  // calculate sum
            DB("delaying...");
            wait(delay);                     // wait delay
            DB("producing...");
            OUT.write(sum);                  // write sum after delay
        }
    }

    SC_CTOR(adder4) {
        SC_THREAD(add);  //thread instead of process         
    }
};

SC_MODULE(driver) {
    unsigned a,b;                //internal data values
    sc_in<bool> clk_in;          //system clock  input
    sc_out<data_t> out_a, out_b; //driver data outputs

    void proc() {
        out_a.write(a);
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

using signal_t = sc_signal<data_t>;

int sc_main(int, char**) {
    sc_clock clk("clk", 5, SC_NS); //system clock
    driver drv("driver");          //test case driver
    adder4 adder("adder");         //device under test

    //connect devices
    signal_t s1, s2, s3;
    drv.clk_in(clk);
    drv.out_a(s1);
    drv.out_b(s2);
    adder.A(s1);
    adder.B(s2);
    adder.OUT(s3);
    
    //trace signals to waveforms.vcd
    sc_trace_file *tf = sc_create_vcd_trace_file("waveforms");
    sc_trace(tf, clk, "clk");
    sc_trace(tf, s1, "A");
    sc_trace(tf, s2, "B");
    sc_trace(tf, s3, "OUT");

    sc_start(40, SC_NS);

    sc_close_vcd_trace_file(tf);

    return 0;
}
